package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type GroupHandler struct {
	groupController *controller.GroupController
}

func NewGroupHandler(groupController *controller.GroupController) *GroupHandler {
	return &GroupHandler{groupController}
}

func (gh *GroupHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/groups", gh.groupController.CreateGroupHandler).Methods("POST")
	r.HandleFunc("/groups", gh.groupController.GetGroupsHandler).Methods("GET")
	r.HandleFunc("/groups/{id}", gh.groupController.GetGroupByIDHandler).Methods("GET")
	r.HandleFunc("/groups/{id}", gh.groupController.UpdateGroupHandler).Methods("PUT")
	r.HandleFunc("/groups/{id}", gh.groupController.DeleteGroupHandler).Methods("DELETE")
}
