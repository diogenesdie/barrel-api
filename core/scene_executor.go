package core

import (
	"barrel-api/internal/mqtt"
	"barrel-api/model"
	"barrel-api/repository"
)

// SceneExecutor executes a scene by sending commands to each device action in order.
type SceneExecutor struct {
	deviceRepo *repository.SmartDeviceRepository
	cmdPub     *mqtt.CommandPublisher
}

func NewSceneExecutor(deviceRepo *repository.SmartDeviceRepository, cmdPub *mqtt.CommandPublisher) *SceneExecutor {
	return &SceneExecutor{deviceRepo: deviceRepo, cmdPub: cmdPub}
}

// ExecuteScene runs all actions in the scene sequentially.
// Failures on individual actions are recorded but do not stop execution.
func (e *SceneExecutor) ExecuteScene(scene *model.Scene, userID uint64) (*model.SceneExecutionResult, error) {
	results := make([]model.SceneActionResult, 0, len(scene.Actions))

	for _, action := range scene.Actions {
		res := model.SceneActionResult{
			DeviceID: action.DeviceID,
			Command:  action.Command,
		}

		device, err := e.deviceRepo.GetSmartDeviceByID(action.DeviceID)
		if err != nil {
			res.Success = false
			res.Error = "device not found"
			results = append(results, res)
			continue
		}

		if device.UserID != userID {
			res.Success = false
			res.Error = "forbidden"
			results = append(results, res)
			continue
		}

		if err := e.cmdPub.PublishDeviceCommand(device.OwnerUsername, device.DeviceID, action.Command); err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
		}

		results = append(results, res)
	}

	return &model.SceneExecutionResult{
		SceneID: scene.ID,
		Actions: results,
	}, nil
}
