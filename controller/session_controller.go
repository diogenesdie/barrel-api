package controller

import (
	"barrel-api/model"
	"barrel-api/repository"
	"encoding/json"
	"net/http"
)

type SessionController struct {
	sessionRepo *repository.SessionRepository
}

func NewSessionController(sessionRepo *repository.SessionRepository) *SessionController {
	return &SessionController{sessionRepo}
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

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Failed to decode request body, verify your data type and fields", http.StatusBadRequest)
		return
	}

	session, err := sc.sessionRepo.Register(&user)

	response := model.Response{
		Message: "User created successfully",
		Data:    session,
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
		response.Data = nil
	}

	w.WriteHeader(response.Status)
	json.NewEncoder(w).Encode(response)
}
