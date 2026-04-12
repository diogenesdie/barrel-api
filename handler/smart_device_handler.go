package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type SmartDeviceHandler struct {
	deviceController *controller.SmartDeviceController
}

func NewSmartDeviceHandler(deviceController *controller.SmartDeviceController) *SmartDeviceHandler {
	return &SmartDeviceHandler{deviceController}
}

func (dh *SmartDeviceHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/devices", dh.deviceController.CreateSmartDeviceHandler).Methods("POST")
	r.HandleFunc("/devices", dh.deviceController.GetSmartDevicesHandler).Methods("GET")
	r.HandleFunc("/devices/{id}", dh.deviceController.GetSmartDeviceByIDHandler).Methods("GET")
	r.HandleFunc("/devices/{id}", dh.deviceController.UpdateSmartDeviceHandler).Methods("PUT")
	r.HandleFunc("/devices/{id}", dh.deviceController.DeleteSmartDeviceHandler).Methods("DELETE")
	r.HandleFunc("/devices/{id}/command", dh.deviceController.CommandHandler).Methods("POST")
	r.HandleFunc("/devices/{id}/buttons", dh.deviceController.GetButtonsHandler).Methods("GET")
	r.HandleFunc("/devices/{id}/buttons", dh.deviceController.UpsertButtonsHandler).Methods("POST")
}
