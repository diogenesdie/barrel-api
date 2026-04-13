package core

import (
	"barrel-api/internal/mqtt"
	"barrel-api/model"
	"barrel-api/repository"
	"fmt"
	"log"
)

// RoutineExecutor executes all actions in a routine sequentially.
type RoutineExecutor struct {
	deviceRepo *repository.SmartDeviceRepository
	sceneRepo  *repository.SceneRepository
	sceneExec  *SceneExecutor
	cmdPub     *mqtt.CommandPublisher
}

func NewRoutineExecutor(
	deviceRepo *repository.SmartDeviceRepository,
	sceneRepo *repository.SceneRepository,
	sceneExec *SceneExecutor,
	cmdPub *mqtt.CommandPublisher,
) *RoutineExecutor {
	return &RoutineExecutor{
		deviceRepo: deviceRepo,
		sceneRepo:  sceneRepo,
		sceneExec:  sceneExec,
		cmdPub:     cmdPub,
	}
}

// ExecuteRoutine runs all actions in the routine in order.
// Errors on individual actions are logged but do not stop execution.
func (e *RoutineExecutor) ExecuteRoutine(routine *model.Routine) error {
	log.Printf("[routine] executing routine %d (%s)", routine.ID, routine.Name)

	for _, action := range routine.Actions {
		if err := e.executeAction(routine.UserID, action); err != nil {
			log.Printf("[routine] action failed (routine=%d type=%s): %v", routine.ID, action.Type, err)
		}
	}
	return nil
}

func (e *RoutineExecutor) executeAction(userID uint64, action model.RoutineAction) error {
	switch action.Type {
	case "device":
		return e.executeDeviceAction(userID, action)
	case "scene":
		return e.executeSceneAction(userID, action)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (e *RoutineExecutor) executeDeviceAction(userID uint64, action model.RoutineAction) error {
	if action.DeviceID == nil || action.Command == nil {
		return fmt.Errorf("device action missing device_id or command")
	}
	device, err := e.deviceRepo.GetSmartDeviceByID(*action.DeviceID)
	if err != nil {
		return fmt.Errorf("device %d not found: %w", *action.DeviceID, err)
	}
	if device.UserID != userID {
		return fmt.Errorf("device %d not owned by user %d", *action.DeviceID, userID)
	}
	return e.cmdPub.PublishDeviceCommand(device.OwnerUsername, device.DeviceID, *action.Command)
}

func (e *RoutineExecutor) executeSceneAction(userID uint64, action model.RoutineAction) error {
	if action.SceneID == nil {
		return fmt.Errorf("scene action missing scene_id")
	}
	scene, err := e.sceneRepo.GetSceneByID(*action.SceneID)
	if err != nil {
		return fmt.Errorf("scene %d not found: %w", *action.SceneID, err)
	}
	if scene.UserID != userID {
		return fmt.Errorf("scene %d not owned by user %d", *action.SceneID, userID)
	}
	_, err = e.sceneExec.ExecuteScene(scene, userID)
	return err
}
