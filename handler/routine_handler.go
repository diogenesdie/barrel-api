package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type RoutineHandler struct {
	routineController *controller.RoutineController
}

func NewRoutineHandler(routineController *controller.RoutineController) *RoutineHandler {
	return &RoutineHandler{routineController}
}

func (rh *RoutineHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/routines", rh.routineController.ListRoutinesHandler).Methods("GET")
	r.HandleFunc("/routines", rh.routineController.CreateRoutineHandler).Methods("POST")
	r.HandleFunc("/routines/{id}", rh.routineController.GetRoutineHandler).Methods("GET")
	r.HandleFunc("/routines/{id}", rh.routineController.UpdateRoutineHandler).Methods("PUT")
	r.HandleFunc("/routines/{id}", rh.routineController.DeleteRoutineHandler).Methods("DELETE")
	r.HandleFunc("/routines/{id}/execute", rh.routineController.ExecuteRoutineHandler).Methods("POST")
}
