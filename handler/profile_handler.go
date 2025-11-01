package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type ProfileHandler struct {
	userController *controller.UserController
}

func NewProfileHandler(userController *controller.UserController) *ProfileHandler {
	return &ProfileHandler{userController}
}

func (ph *ProfileHandler) RegisterRoutes(mux *mux.Router) {
	mux.HandleFunc("/profile", ph.userController.UpdateUserProfileHandler).Methods("PATCH")
}
