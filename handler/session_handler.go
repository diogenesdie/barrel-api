package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type SessionHandler struct {
	sessionController *controller.SessionController
}

func NewSessionHandler(sessionController *controller.SessionController) *SessionHandler {
	return &SessionHandler{sessionController}
}

func (sh *SessionHandler) RegisterRoutes(mux *mux.Router) {
	mux.HandleFunc("/login", sh.sessionController.Login).Methods("POST")
	mux.HandleFunc("/register", sh.sessionController.Register).Methods("POST")
}
