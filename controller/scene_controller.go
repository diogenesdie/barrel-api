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

// SceneExecutorInterface decouples the controller from the concrete executor.
type SceneExecutorInterface interface {
	ExecuteScene(scene *model.Scene, userID uint64) (*model.SceneExecutionResult, error)
}

// SceneController handles HTTP requests for the scenes resource.
type SceneController struct {
	sceneRepo repository.SceneRepositoryInterface
	executor  SceneExecutorInterface
}

func NewSceneController(sceneRepo repository.SceneRepositoryInterface, executor SceneExecutorInterface) *SceneController {
	return &SceneController{sceneRepo: sceneRepo, executor: executor}
}

// ListScenesHandler handles GET /api/v1/scenes
func (sc *SceneController) ListScenesHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	scenes, err := sc.sceneRepo.GetScenesByUser(userID)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to list scenes", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", scenes)
}

// CreateSceneHandler handles POST /api/v1/scenes
func (sc *SceneController) CreateSceneHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var scene model.Scene
	if err := json.NewDecoder(r.Body).Decode(&scene); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	scene.UserID = userID

	if scene.Name == "" {
		writeResponse(w, http.StatusBadRequest, "Field 'name' is required", nil)
		return
	}

	id, err := sc.sceneRepo.CreateScene(&scene)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to create scene", nil)
		return
	}
	scene.ID = id

	writeResponse(w, http.StatusCreated, "Scene created successfully", scene)
}

// GetSceneHandler handles GET /api/v1/scenes/{id}
func (sc *SceneController) GetSceneHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid scene ID", nil)
		return
	}

	scene, err := sc.sceneRepo.GetSceneByID(id)
	if err == repository.ErrSceneNotFound {
		writeResponse(w, http.StatusNotFound, "Scene not found", nil)
		return
	}
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get scene", nil)
		return
	}

	if scene.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	writeResponse(w, http.StatusOK, "OK", scene)
}

// UpdateSceneHandler handles PUT /api/v1/scenes/{id}
func (sc *SceneController) UpdateSceneHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid scene ID", nil)
		return
	}

	existing, err := sc.sceneRepo.GetSceneByID(id)
	if err == repository.ErrSceneNotFound {
		writeResponse(w, http.StatusNotFound, "Scene not found", nil)
		return
	}
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get scene", nil)
		return
	}
	if existing.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var updated model.Scene
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeResponse(w, http.StatusBadRequest, "Failed to decode request body", nil)
		return
	}
	updated.ID = id
	updated.UserID = userID

	if err := sc.sceneRepo.UpdateScene(&updated); err == repository.ErrSceneNotFound {
		writeResponse(w, http.StatusNotFound, "Scene not found", nil)
		return
	} else if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to update scene", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Scene updated successfully", updated)
}

// DeleteSceneHandler handles DELETE /api/v1/scenes/{id}
func (sc *SceneController) DeleteSceneHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid scene ID", nil)
		return
	}

	if err := sc.sceneRepo.DeleteScene(id, userID); err == repository.ErrSceneNotFound {
		writeResponse(w, http.StatusNotFound, "Scene not found", nil)
		return
	} else if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to delete scene", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Scene deleted successfully", nil)
}

// ExecuteSceneHandler handles POST /api/v1/scenes/{id}/execute
func (sc *SceneController) ExecuteSceneHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetParsedUserId(r.Header.Get("user_id"))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	id, err := parseIDParam(r, "id")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid scene ID", nil)
		return
	}

	scene, err := sc.sceneRepo.GetSceneByID(id)
	if err == repository.ErrSceneNotFound {
		writeResponse(w, http.StatusNotFound, "Scene not found", nil)
		return
	}
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to get scene", nil)
		return
	}
	if scene.UserID != userID {
		writeResponse(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	result, err := sc.executor.ExecuteScene(scene, userID)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to execute scene", nil)
		return
	}

	writeResponse(w, http.StatusOK, "Scene executed", result)
}

func parseIDParam(r *http.Request, param string) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)[param], 10, 64)
}
