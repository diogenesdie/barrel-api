package controller

import (
	"barrel-api/auth"
	"barrel-api/model"
	"barrel-api/repository"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type SmartDeviceController struct {
	deviceRepo           *repository.SmartDeviceRepository
	groupRepo            *repository.GroupRepository
	deviceShareRepo      *repository.DeviceShareRepository
	smartDeviceShareRepo *repository.SmartDeviceShareRepository
}

func NewSmartDeviceController(deviceRepo *repository.SmartDeviceRepository, groupRepo *repository.GroupRepository, deviceShareRepo *repository.DeviceShareRepository, smartDeviceShareRepo *repository.SmartDeviceShareRepository) *SmartDeviceController {
	return &SmartDeviceController{deviceRepo: deviceRepo, groupRepo: groupRepo, deviceShareRepo: deviceShareRepo, smartDeviceShareRepo: smartDeviceShareRepo}
}

func (dc *SmartDeviceController) CreateSmartDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var device model.SmartDevice
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	device.UserID = userID

	// valida se o grupo pertence ao user
	if device.GroupID != nil {
		group, err := dc.groupRepo.GetGroupByID(*device.GroupID)

		if err != nil || group.UserID != userID {
			writeResponse(w, http.StatusForbidden, "Invalid group", nil)
			return
		}
	}

	if deviceId, err := dc.deviceRepo.CreateSmartDevice(&device); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to create device", nil)
		return
	} else {
		device.ID = deviceId
	}

	writeResponse(w, http.StatusCreated, "Device created successfully", device)
}

func (dc *SmartDeviceController) GetSmartDevicesHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		print(err.Error())
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	devices, err := dc.deviceRepo.GetSmartDevicesByUser(userID)
	if err != nil {
		print(err.Error())
		writeResponse(w, http.StatusInternalServerError, "Failed to get devices", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", devices)
}

func (dc *SmartDeviceController) GetSmartDeviceByIDHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id := mux.Vars(r)["id"]
	deviceID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid device ID", nil)
		return
	}

	device, err := dc.deviceRepo.GetSmartDeviceByID(deviceID)
	if err != nil {
		if err == repository.ErrSmartDeviceNotFound {
			writeResponse(w, http.StatusNotFound, "Device not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get device", nil)
		return
	}

	if device.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", device)
}

func (dc *SmartDeviceController) UpdateSmartDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id := mux.Vars(r)["id"]

	deviceID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid device ID", nil)
		return
	}

	device, err := dc.deviceRepo.GetSmartDeviceByID(deviceID)
	if err != nil {
		if err == repository.ErrSmartDeviceNotFound {
			writeResponse(w, http.StatusNotFound, "Device not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get device", nil)
		return
	}

	// Se o usuário for dono → update normal em smart_devices
	if device.UserID == userID {
		if err := json.NewDecoder(r.Body).Decode(device); err != nil {
			writeResponse(w, http.StatusBadRequest, "Failed to decode body", nil)
			return
		}
		device.UserID = userID

		if device.GroupID != nil {
			group, err := dc.groupRepo.GetGroupByID(*device.GroupID)
			if err != nil || group.UserID != userID {
				writeResponse(w, http.StatusForbidden, "Invalid group", nil)
				return
			}
		}

		if err := dc.deviceRepo.UpdateSmartDevice(device); err != nil {
			writeResponse(w, http.StatusInternalServerError, "Failed to update device", nil)
			return
		}

		writeResponse(w, http.StatusOK, "Device updated successfully", device)
		return
	}

	share, err := dc.deviceShareRepo.GetActiveShareByDeviceAndUser(deviceID, userID)
	if err != nil {
		if err == repository.ErrDeviceShareNotFound {
			writeResponse(w, http.StatusForbidden, "Forbidden", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to check device share", nil)
		return
	}

	var shareUpdate model.SmartDeviceShare
	if err := json.NewDecoder(r.Body).Decode(&shareUpdate); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode body", nil)
		return
	}

	//print as json
	shareUpdateJson, _ := json.Marshal(shareUpdate)
	print(string(shareUpdateJson))

	shareUpdate.DeviceShareID = share.ID
	shareUpdate.DeviceID = deviceID
	shareUpdate.UserID = userID

	if err := dc.smartDeviceShareRepo.UpsertSmartDeviceShare(&shareUpdate); err != nil {
		print(err.Error())
		writeResponse(w, http.StatusInternalServerError, "Failed to update shared device", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Shared device updated successfully", shareUpdate)
}

func (dc *SmartDeviceController) DeleteSmartDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id := mux.Vars(r)["id"]

	deviceID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid device ID", nil)
		return
	}

	device, err := dc.deviceRepo.GetSmartDeviceByID(deviceID)
	if err != nil {
		if err == repository.ErrSmartDeviceNotFound {
			writeResponse(w, http.StatusNotFound, "Device not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get device", nil)
		return
	}

	if device.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	if err := dc.deviceRepo.DeleteSmartDevice(id); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to delete device", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Device deleted successfully", nil)
}
