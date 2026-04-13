package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type SceneHandler struct {
	sceneController *controller.SceneController
}

func NewSceneHandler(sceneController *controller.SceneController) *SceneHandler {
	return &SceneHandler{sceneController}
}

func (sh *SceneHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/scenes", sh.sceneController.ListScenesHandler).Methods("GET")
	r.HandleFunc("/scenes", sh.sceneController.CreateSceneHandler).Methods("POST")
	r.HandleFunc("/scenes/{id}", sh.sceneController.GetSceneHandler).Methods("GET")
	r.HandleFunc("/scenes/{id}", sh.sceneController.UpdateSceneHandler).Methods("PUT")
	r.HandleFunc("/scenes/{id}", sh.sceneController.DeleteSceneHandler).Methods("DELETE")
	r.HandleFunc("/scenes/{id}/execute", sh.sceneController.ExecuteSceneHandler).Methods("POST")
}
