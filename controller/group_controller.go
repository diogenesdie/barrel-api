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

type GroupController struct {
	groupRepo *repository.GroupRepository
}

func NewGroupController(groupRepo *repository.GroupRepository) *GroupController {
	return &GroupController{groupRepo: groupRepo}
}

func (gc *GroupController) CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var group model.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	group.UserID = userID

	if err := gc.groupRepo.CreateGroup(&group); err != nil {
		print(err.Error())
		writeResponse(w, http.StatusInternalServerError, "Failed to create group", nil)
		return
	}

	writeResponse(w, http.StatusCreated, "Group created successfully", group)
}

func (gc *GroupController) GetGroupsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	groups, err := gc.groupRepo.GetGroupsByUser(userID)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get groups", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", groups)
}

func (gc *GroupController) GetGroupByIDHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid group ID", nil)
		return
	}

	group, err := gc.groupRepo.GetGroupByID(id)
	if err != nil {
		if err == repository.ErrGroupNotFound {
			writeResponse(w, http.StatusNotFound, "Group not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get group", nil)
		return
	}

	if group.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", group)
}

func (gc *GroupController) UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid group ID", nil)
		return
	}

	group, err := gc.groupRepo.GetGroupByID(id)
	if err != nil {
		if err == repository.ErrGroupNotFound {
			writeResponse(w, http.StatusNotFound, "Group not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get group", nil)
		return
	}

	if group.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(group); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode body", nil)
		return
	}
	group.ID = id
	group.UserID = userID

	if err := gc.groupRepo.UpdateGroup(group); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to update group", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Group updated successfully", group)
}

func (gc *GroupController) DeleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid group ID", nil)
		return
	}

	group, err := gc.groupRepo.GetGroupByID(id)
	if err != nil {
		if err == repository.ErrGroupNotFound {
			writeResponse(w, http.StatusNotFound, "Group not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "Failed to get group", nil)
		return
	}

	if group.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	if err := gc.groupRepo.DeleteGroup(id); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to delete group", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Group deleted successfully", nil)
}
