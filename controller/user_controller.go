package controller

import (
	"barrel-api/internal/mqtt"
	"barrel-api/model"
	"barrel-api/repository"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type UserController struct {
	userRepo *repository.UserRepository
	mqttProv mqtt.Provisioner
}

func NewUserController(userRepo *repository.UserRepository, prov mqtt.Provisioner) *UserController {
	return &UserController{userRepo: userRepo, mqttProv: prov}
}

func (uc *UserController) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	if user.Username == "" || user.Password == nil {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}
	rawPass := *user.Password

	// 1) cria no banco
	if err := uc.userRepo.CreateUser(&user); err != nil {
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// 1) cria client no Mosquitto
	if err := uc.mqttProv.CreateUser(ctx, mqtt.User{
		Username: user.Username,
		Password: rawPass,
	}); err != nil {
		log.Printf("Failed to provision MQTT user: %v", err)
		_ = uc.userRepo.DeleteUser(user.ID)
		http.Error(w, "Failed to provision MQTT user", http.StatusBadGateway)
		return
	}

	// 2) cria role com o nome do usuário
	role := "role_" + user.Username
	if err := uc.mqttProv.CreateRole(ctx, role); err != nil {
		log.Printf("Failed to create role: %v", err)
	}

	// 3) dá permissão para publish/subscribe em users/<username>/#
	topic := "users/" + user.Username + "/#"
	_ = uc.mqttProv.AddRoleACL(ctx, role, "subscribePattern", topic)
	_ = uc.mqttProv.AddRoleACL(ctx, role, "publishClientSend", topic)

	// 4) vincula usuário à role
	_ = uc.mqttProv.AddClientRole(ctx, user.Username, role)

	w.WriteHeader(http.StatusCreated)
}

func (uc *UserController) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := uc.userRepo.GetUserByID(id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

func (uc *UserController) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := uc.userRepo.GetUsers()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}
	if len(users) == 0 {
		http.Error(w, "No users found", http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(users)
}

func (uc *UserController) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	user.ID = id

	var newPass string
	if user.Password != nil {
		newPass = *user.Password
	}

	if err := uc.userRepo.UpdateUser(&user); err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	if newPass != "" {
		ctx := context.Background()
		if err := uc.mqttProv.UpdatePassword(ctx, user.Username, newPass); err != nil {
			log.Printf("Failed to update MQTT password for user %s: %v", user.Username, err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (uc *UserController) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	u, _ := uc.userRepo.GetUserByID(id)

	if err := uc.userRepo.DeleteUser(id); err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if u != nil {
		ctx := context.Background()
		if err := uc.mqttProv.DeleteUser(ctx, u.Username); err != nil {
			log.Printf("Failed to delete MQTT user %s: %v", u.Username, err)
		}
		// remove a role vinculada
		role := "role_" + u.Username
		if err := uc.mqttProv.DeleteRole(ctx, role); err != nil {
			log.Printf("Failed to delete MQTT role %s: %v", role, err)
		}
	}

	w.WriteHeader(http.StatusOK)
}
