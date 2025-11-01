package controller

import (
	"barrel-api/internal/mqtt"
	"barrel-api/model"
	"barrel-api/repository"
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type SessionController struct {
	sessionRepo *repository.SessionRepository
	groupRepo   *repository.GroupRepository
	mqttProv    mqtt.Provisioner
}

func NewSessionController(
	sessionRepo *repository.SessionRepository,
	groupRepo *repository.GroupRepository,
	mqttProv mqtt.Provisioner,
) *SessionController {
	return &SessionController{
		sessionRepo: sessionRepo,
		groupRepo:   groupRepo,
		mqttProv:    mqttProv,
	}
}

func (sc *SessionController) Login(w http.ResponseWriter, r *http.Request) {
	var login model.Login

	err := json.NewDecoder(r.Body).Decode(&login)

	if err != nil {
		http.Error(w, "Failed to decode request body, verify your data type and fields", http.StatusBadRequest)
		return
	}

	session, err := sc.sessionRepo.Login(&login)

	response := model.Response{
		Message: "OK",
		Data:    session,
		Code:    0,
		Status:  http.StatusOK,
	}

	if err != nil {
		switch err {
		case repository.ErrSessionNotFound:
			response.Status = http.StatusNotFound
		case repository.ErrSessionExpired:
			response.Status = http.StatusUnauthorized
		case repository.ErrSessionInactive:
			response.Status = http.StatusUnauthorized
		case repository.ErrInvalidPassword:
			response.Status = http.StatusUnauthorized
		case repository.ErrUserNotFound:
			response.Status = http.StatusNotFound
		case repository.ErrUnauthorized:
			response.Status = http.StatusUnauthorized
		case repository.ErrGenerateToken:
			response.Status = http.StatusInternalServerError
		case repository.ErrUpdateToken:
			response.Status = http.StatusInternalServerError
		default:
			response.Status = http.StatusInternalServerError
		}

		response.Message = err.Error()
		response.Data = nil
	}

	w.WriteHeader(response.Status)
	json.NewEncoder(w).Encode(response)
}

func (sc *SessionController) Register(w http.ResponseWriter, r *http.Request) {
	var user model.User

	// 1. decode body
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Failed to decode request body, verify your data type and fields", http.StatusBadRequest)
		return
	}

	// segurança básica: username e password obrigatórios
	if user.Username == "" || user.Password == nil {
		resp := model.Response{
			Message: "username and password are required",
			Data:    nil,
			Code:    0,
			Status:  http.StatusBadRequest,
		}
		w.WriteHeader(resp.Status)
		json.NewEncoder(w).Encode(resp)
		return
	}

	rawPass := *user.Password // guarda senha crua pro MQTT depois

	// 2. chama Register no repo (que cria user, cria sessão etc.)
	regResult, err := sc.sessionRepo.Register(&user)

	// prepara resposta padrão
	response := model.Response{
		Message: "User created successfully",
		Data:    nil,
		Code:    0,
		Status:  http.StatusCreated,
	}

	if err != nil {
		switch err {
		case repository.ErrUserAlreadyExists:
			response.Status = http.StatusConflict
			response.Message = "User already exists"
		default:
			response.Status = http.StatusInternalServerError
			response.Message = "Internal server error"
		}
		w.WriteHeader(response.Status)
		json.NewEncoder(w).Encode(response)
		return
	}

	// até aqui: o user já foi criado no banco e já tem sessão válida
	// agora vem a parte que antes só existia no CreateUserHandler:
	userID := regResult.UserID
	username := regResult.Username

	// 3. cria grupos padrão para o usuário
	iconHouse := "house"
	defaultGroup := model.Group{
		Name:      "Casa",
		UserID:    userID,
		Icon:      &iconHouse,
		IsDefault: true,
		Position:  0,
	}
	if err := sc.groupRepo.CreateGroup(&defaultGroup); err != nil {
		log.Printf("Failed to create default group for user %d: %v", userID, err)
		// Se falhar grupo, eu NÃO vou invalidar o user/sessão porque tu não derruba isso nem no UserController hoje.
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
	if err := sc.groupRepo.CreateGroup(&shareGroup); err != nil {
		log.Printf("Failed to create share group for user %d: %v", userID, err)
	}

	// 4. provisiona MQTT
	ctx := context.Background()

	if err := sc.mqttProv.CreateUser(ctx, mqtt.User{
		Username: username,
		Password: rawPass,
	}); err != nil {
		log.Printf("Failed to provision MQTT user: %v", err)
		// mesma ideia: se falha MQTT agora, não apago o user pq no fluxo antigo tu tb não fazia rollback completo
	}

	role := "role_" + username
	if err := sc.mqttProv.CreateRole(ctx, role); err != nil {
		log.Printf("Failed to create role: %v", err)
	}

	topic := "users/" + username + "/#"
	_ = sc.mqttProv.AddRoleACL(ctx, role, "subscribePattern", topic)
	_ = sc.mqttProv.AddRoleACL(ctx, role, "publishClientSend", topic)
	_ = sc.mqttProv.AddClientRole(ctx, username, role)

	// 5. resposta final pro app (idêntica ao login): a Session
	response.Data = regResult

	w.WriteHeader(response.Status)
	json.NewEncoder(w).Encode(response)
}
