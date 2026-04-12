package controller

import (
	"barrel-api/auth"
	"barrel-api/internal/mqtt"
	"barrel-api/model"
	"barrel-api/repository"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
)

// commandAllowList valida os comandos aceitos pelo endpoint /command.
var commandAllowList = regexp.MustCompile(`^(on|off|pulse|brightness:\d{1,3}|send:[A-Z0-9_]+)$`)

type SmartDeviceController struct {
	deviceRepo           *repository.SmartDeviceRepository
	groupRepo            *repository.GroupRepository
	deviceShareRepo      *repository.DeviceShareRepository
	smartDeviceShareRepo *repository.SmartDeviceShareRepository
	buttonRepo           *repository.DeviceButtonRepository
	cmdPub               *mqtt.CommandPublisher
}

func NewSmartDeviceController(
	deviceRepo *repository.SmartDeviceRepository,
	groupRepo *repository.GroupRepository,
	deviceShareRepo *repository.DeviceShareRepository,
	smartDeviceShareRepo *repository.SmartDeviceShareRepository,
	buttonRepo *repository.DeviceButtonRepository,
	cmdPub *mqtt.CommandPublisher,
) *SmartDeviceController {
	return &SmartDeviceController{
		deviceRepo:           deviceRepo,
		groupRepo:            groupRepo,
		deviceShareRepo:      deviceShareRepo,
		smartDeviceShareRepo: smartDeviceShareRepo,
		buttonRepo:           buttonRepo,
		cmdPub:               cmdPub,
	}
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

// CommandHandler envia um comando para o dispositivo via MQTT.
// POST /api/v1/devices/{id}/command
// Body: {"command": "on"|"off"|"brightness:50"|"send:BTN_1"}
func (dc *SmartDeviceController) CommandHandler(w http.ResponseWriter, r *http.Request) {
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

	var body struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Command == "" {
		writeResponse(w, http.StatusBadRequest, "Campo 'command' obrigatório", nil)
		return
	}

	if !commandAllowList.MatchString(body.Command) {
		writeResponse(w, http.StatusUnprocessableEntity, "Comando inválido", nil)
		return
	}

	if err := dc.cmdPub.PublishDeviceCommand(device.OwnerUsername, device.DeviceID, body.Command); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Falha ao enviar comando: "+err.Error(), nil)
		return
	}

	// Atualiza estado no banco para switch/dimmer
	newState := ""
	switch body.Command {
	case "on":
		newState = "on"
	case "off":
		newState = "off"
	default:
		if len(body.Command) > 11 && body.Command[:11] == "brightness:" {
			newState = "on"
		}
	}
	if newState != "" {
		_ = dc.deviceRepo.UpdateSmartDeviceState(deviceID, newState)
	}

	writeResponse(w, http.StatusOK, "Comando enviado", map[string]string{"command": body.Command})
}

// GetButtonsHandler retorna os botões IR/RF de um dispositivo.
// GET /api/v1/devices/{id}/buttons
func (dc *SmartDeviceController) GetButtonsHandler(w http.ResponseWriter, r *http.Request) {
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

	buttons, err := dc.buttonRepo.GetButtonsByDeviceID(deviceID)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get buttons", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", buttons)
}

// UpsertButtonsHandler sincroniza os botões de um dispositivo IR/RF.
// POST /api/v1/devices/{id}/buttons
// Body: {"buttons": [...]}
func (dc *SmartDeviceController) UpsertButtonsHandler(w http.ResponseWriter, r *http.Request) {
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

	var req model.UpsertDeviceButtonsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode body", nil)
		return
	}

	if err := dc.buttonRepo.UpsertButtons(deviceID, req.Buttons); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to upsert buttons", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Buttons synced", nil)
}
