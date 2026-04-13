package controller

import (
	"barrel-api/auth"
	"barrel-api/model"
	"barrel-api/repository"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// RoutineExecutorInterface decouples the controller from the concrete executor.
type RoutineExecutorInterface interface {
	ExecuteRoutine(routine *model.Routine) error
}

// RoutineController handles HTTP requests for the routines resource.
type RoutineController struct {
	routineRepo repository.RoutineRepositoryInterface
	executor    RoutineExecutorInterface
}

func NewRoutineController(routineRepo repository.RoutineRepositoryInterface, executor RoutineExecutorInterface) *RoutineController {
	return &RoutineController{routineRepo: routineRepo, executor: executor}
}

// ListRoutinesHandler handles GET /api/v1/routines
func (rc *RoutineController) ListRoutinesHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	routines, err := rc.routineRepo.GetRoutinesByUser(userID)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to list routines", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", routines)
}

// CreateRoutineHandler handles POST /api/v1/routines
func (rc *RoutineController) CreateRoutineHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var routine model.Routine
	if err := json.NewDecoder(r.Body).Decode(&routine); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	routine.UserID = userID

	if routine.Name == "" {
		writeResponse(w, http.StatusBadRequest, "Field 'name' is required", nil)
		return
	}
	if routine.Trigger.Type != "device" && routine.Trigger.Type != "schedule" {
		writeResponse(w, http.StatusBadRequest, "Field 'trigger.type' must be 'device' or 'schedule'", nil)
		return
	}

	id, err := rc.routineRepo.CreateRoutine(&routine)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to create routine", nil)
		return
	}
	routine.ID = id

	writeResponse(w, http.StatusCreated, "Routine created successfully", routine)
}

// GetRoutineHandler handles GET /api/v1/routines/{id}
func (rc *RoutineController) GetRoutineHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid routine ID", nil)
		return
	}

	routine, err := rc.routineRepo.GetRoutineByID(id)
	if err == repository.ErrRoutineNotFound {
		writeResponse(w, http.StatusNotFound, "Routine not found", nil)
		return
	}
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get routine", nil)
		return
	}
	if routine.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", routine)
}

// UpdateRoutineHandler handles PUT /api/v1/routines/{id}
func (rc *RoutineController) UpdateRoutineHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid routine ID", nil)
		return
	}

	existing, err := rc.routineRepo.GetRoutineByID(id)
	if err == repository.ErrRoutineNotFound {
		writeResponse(w, http.StatusNotFound, "Routine not found", nil)
		return
	}
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get routine", nil)
		return
	}
	if existing.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var updated model.Routine
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	updated.ID = id
	updated.UserID = userID

	if err := rc.routineRepo.UpdateRoutine(&updated); err == repository.ErrRoutineNotFound {
		writeResponse(w, http.StatusNotFound, "Routine not found", nil)
		return
	} else if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to update routine", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Routine updated successfully", updated)
}

// DeleteRoutineHandler handles DELETE /api/v1/routines/{id}
func (rc *RoutineController) DeleteRoutineHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid routine ID", nil)
		return
	}

	if err := rc.routineRepo.DeleteRoutine(id, userID); err == repository.ErrRoutineNotFound {
		writeResponse(w, http.StatusNotFound, "Routine not found", nil)
		return
	} else if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to delete routine", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Routine deleted successfully", nil)
}

// ExecuteRoutineHandler handles POST /api/v1/routines/{id}/execute
func (rc *RoutineController) ExecuteRoutineHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid routine ID", nil)
		return
	}

	routine, err := rc.routineRepo.GetRoutineByID(id)
	if err == repository.ErrRoutineNotFound {
		writeResponse(w, http.StatusNotFound, "Routine not found", nil)
		return
	}
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get routine", nil)
		return
	}
	if routine.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	if err := rc.executor.ExecuteRoutine(routine); err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to execute routine", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Routine executed", nil)
}

// Ensure mux import is used (parseIDParam is defined in scene_controller.go)
var _ = mux.Vars
