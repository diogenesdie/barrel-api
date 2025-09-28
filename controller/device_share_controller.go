package controller

import (
	"barrel-api/auth"
	"barrel-api/internal/mqtt"
	"barrel-api/model"
	"barrel-api/repository"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type DeviceShareController struct {
	shareRepo  *repository.DeviceShareRepository
	deviceRepo *repository.SmartDeviceRepository
	groupRepo  *repository.GroupRepository
	userRepo   *repository.UserRepository
	mqttProv   mqtt.Provisioner
}

func NewDeviceShareController(shareRepo *repository.DeviceShareRepository, deviceRepo *repository.SmartDeviceRepository, groupRepo *repository.GroupRepository, userRepo *repository.UserRepository, mqttProv mqtt.Provisioner) *DeviceShareController {
	return &DeviceShareController{shareRepo, deviceRepo, groupRepo, userRepo, mqttProv}
}

func (c *DeviceShareController) CreateShareHandler(w http.ResponseWriter, r *http.Request) {
	ownerID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var ds model.DeviceShare
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode body", nil)
		return
	}
	ds.OwnerID = ownerID

	if ds.DeviceID == nil && ds.GroupID == nil {
		writeResponse(w, http.StatusBadRequest, "device_id or group_id required", nil)
		return
	}

	if ds.DeviceID != nil {
		device, err := c.deviceRepo.GetSmartDeviceByID(*ds.DeviceID)
		if err != nil || device.UserID != ownerID {
			writeResponse(w, http.StatusForbidden, "Device does not belong to user", nil)
			return
		}
	}

	if ds.GroupID != nil {
		group, err := c.groupRepo.GetGroupByID(*ds.GroupID)
		if err != nil || group.UserID != ownerID {
			writeResponse(w, http.StatusForbidden, "Group does not belong to user", nil)
			return
		}
	}

	exists, err := c.shareRepo.ExistsActiveShare(ds.DeviceID, ds.GroupID, ds.SharedWithID)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to check existing shares", nil)
		return
	}
	if exists {
		writeResponse(w, http.StatusConflict, "Resource already shared with this user", nil)
		return
	}

	if err := c.shareRepo.Create(&ds); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to create share", nil)
		return
	}

	writeResponse(w, http.StatusCreated, "Share created successfully (pending)", nil)
}

func (c *DeviceShareController) AcceptShareHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetParsedUserId(r.Header.Get("user_id"))
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	share, err := c.shareRepo.GetByID(id)
	if err != nil {
		writeResponse(w, http.StatusNotFound, "Share not found", nil)
		return
	}

	if share.SharedWithID != userID {
		writeResponse(w, http.StatusForbidden, "You cannot accept this share", nil)
		return
	}
	if share.Status != "P" {
		writeResponse(w, http.StatusBadRequest, "Share is not pending", nil)
		return
	}

	if err := c.shareRepo.UpdateStatus(share.ID, "A"); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to accept share", nil)
		return
	}

	ctx := context.Background()
	sharedWith, _ := c.userRepo.GetUserByID(share.SharedWithID)
	owner, _ := c.userRepo.GetUserByID(share.OwnerID)

	role := "role_" + sharedWith.Username
	topic := fmt.Sprintf("users/%s/#", owner.Username)

	_ = c.mqttProv.AddRoleACL(ctx, role, "subscribePattern", topic)
	_ = c.mqttProv.AddRoleACL(ctx, role, "publishClientSend", topic)

	writeResponse(w, http.StatusOK, "Share accepted", nil)
}

func (c *DeviceShareController) RevokeShareHandler(w http.ResponseWriter, r *http.Request) {
	ownerID, _ := auth.GetParsedUserId(r.Header.Get("user_id"))
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	share, err := c.shareRepo.GetByID(id)
	if err != nil {
		writeResponse(w, http.StatusNotFound, "Share not found", nil)
		return
	}

	if share.OwnerID != ownerID {
		writeResponse(w, http.StatusForbidden, "You cannot revoke this share", nil)
		return
	}
	if share.Status != "A" {
		writeResponse(w, http.StatusBadRequest, "Share is not active", nil)
		return
	}

	if err := c.shareRepo.UpdateStatus(share.ID, "R"); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to revoke share", nil)
		return
	}

	ctx := context.Background()
	sharedWith, _ := c.userRepo.GetUserByID(share.SharedWithID)
	owner, _ := c.userRepo.GetUserByID(share.OwnerID)

	role := "role_" + sharedWith.Username
	topic := fmt.Sprintf("users/%s/#", owner.Username)

	_ = c.mqttProv.RemoveRoleACL(ctx, role, "subscribePattern", topic)
	_ = c.mqttProv.RemoveRoleACL(ctx, role, "publishClientSend", topic)

	writeResponse(w, http.StatusOK, "Share revoked", nil)
}

func (c *DeviceShareController) GetSharesHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetParsedUserId(r.Header.Get("user_id"))

	shares, err := c.shareRepo.GetByUser(userID)
	if err != nil {
		print(err.Error())
		writeResponse(w, http.StatusInternalServerError, "Failed to get shares", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", shares)
}
