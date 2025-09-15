package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type DeviceShareHandler struct {
	ctrl *controller.DeviceShareController
}

func NewDeviceShareHandler(ctrl *controller.DeviceShareController) *DeviceShareHandler {
	return &DeviceShareHandler{ctrl}
}

func (h *DeviceShareHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/shares", h.ctrl.CreateShareHandler).Methods("POST")
	r.HandleFunc("/shares", h.ctrl.GetSharesHandler).Methods("GET")
	r.HandleFunc("/shares/accept", h.ctrl.AcceptShareHandler).Methods("POST")
	r.HandleFunc("/shares/revoke", h.ctrl.RevokeShareHandler).Methods("POST")
}
