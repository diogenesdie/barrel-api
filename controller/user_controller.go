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
	userRepo  *repository.UserRepository
	groupRepo *repository.GroupRepository
	mqttProv  mqtt.Provisioner
}

func NewUserController(userRepo *repository.UserRepository, groupRepo *repository.GroupRepository, prov mqtt.Provisioner) *UserController {
	return &UserController{userRepo: userRepo, groupRepo: groupRepo, mqttProv: prov}
}

func (uc *UserController) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}

	if user.Username == "" || user.Password == nil {
		writeResponse(w, http.StatusBadRequest, "username and password are required", nil)
		return
	}
	rawPass := *user.Password

	if userID, err := uc.userRepo.CreateUser(&user); err != nil {
		log.Printf("Failed to create user: %v", err)
		writeResponse(w, http.StatusInternalServerError, "Failed to create user", nil)
		return
	} else {
		icon := "house"
		defaultGroup := model.Group{
			Name:      "Casa",
			UserID:    userID,
			Icon:      &icon,
			IsDefault: true,
			Position:  0,
		}

		if err := uc.groupRepo.CreateGroup(&defaultGroup); err != nil {
			log.Printf("Failed to create group: %v", err)
			writeResponse(w, http.StatusInternalServerError, "Failed to create group", nil)
			return
		}

		shareIcon := "share"
		shareGroup := model.Group{
			Name:         "Compartilhados comigo",
			UserID:       userID,
			IsDefault:    false,
			Icon:         &shareIcon,
			IsShareGroup: true,
			Position:     1,
		}

		if err := uc.groupRepo.CreateGroup(&shareGroup); err != nil {
			log.Printf("Failed to create share group: %v", err)
			writeResponse(w, http.StatusInternalServerError, "Failed to create share group", nil)
			return
		}
	}

	ctx := context.Background()
	if err := uc.mqttProv.CreateUser(ctx, mqtt.User{
		Username: user.Username,
		Password: rawPass,
	}); err != nil {
		log.Printf("Failed to provision MQTT user: %v", err)
		_ = uc.userRepo.DeleteUser(user.ID)
		writeResponse(w, http.StatusBadGateway, "Failed to provision MQTT user", nil)
		return
	}

	role := "role_" + user.Username
	if err := uc.mqttProv.CreateRole(ctx, role); err != nil {
		log.Printf("Failed to create role: %v", err)
	}

	topic := "users/" + user.Username + "/#"
	_ = uc.mqttProv.AddRoleACL(ctx, role, "subscribePattern", topic)
	_ = uc.mqttProv.AddRoleACL(ctx, role, "publishClientSend", topic)
	_ = uc.mqttProv.AddClientRole(ctx, user.Username, role)

	writeResponse(w, http.StatusCreated, "User created successfully", user)
}

func (uc *UserController) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	user, err := uc.userRepo.GetUserByID(id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			writeResponse(w, http.StatusNotFound, "User not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get user", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", user)
}

func (uc *UserController) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := uc.userRepo.GetUsers()
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get users", nil)
		return
	}
	if len(users) == 0 {
		writeResponse(w, http.StatusNotFound, "No users found", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", users)
}

func (uc *UserController) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	user.ID = id

	var newPass string
	if user.Password != nil {
		newPass = *user.Password
	}

	if err := uc.userRepo.UpdateUser(&user); err != nil {
		if err == repository.ErrUserNotFound {
			writeResponse(w, http.StatusNotFound, "User not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to update user", nil)
		return
	}

	if newPass != "" {
		ctx := context.Background()
		if err := uc.mqttProv.UpdatePassword(ctx, user.Username, newPass); err != nil {
			log.Printf("Failed to update MQTT password for user %s: %v", user.Username, err)
		}
	}

	writeResponse(w, http.StatusOK, "User updated successfully", user)
}

func (uc *UserController) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	u, _ := uc.userRepo.GetUserByID(id)

	if err := uc.userRepo.DeleteUser(id); err != nil {
		if err == repository.ErrUserNotFound {
			writeResponse(w, http.StatusNotFound, "User not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to delete user", nil)
		return
	}

	if u != nil {
		ctx := context.Background()
		if err := uc.mqttProv.DeleteUser(ctx, u.Username); err != nil {
			log.Printf("Failed to delete MQTT user %s: %v", u.Username, err)
		}
		role := "role_" + u.Username
		if err := uc.mqttProv.DeleteRole(ctx, role); err != nil {
			log.Printf("Failed to delete MQTT role %s: %v", role, err)
		}
	}

	writeResponse(w, http.StatusOK, "User deleted successfully", nil)
}

// helper para resposta padronizada
func writeResponse(w http.ResponseWriter, status int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := model.Response{
		Message: message,
		Data:    data,
		Code:    0,
		Status:  status,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
