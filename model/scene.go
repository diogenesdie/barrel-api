package model

import "time"

// Scene represents a named collection of device actions executed atomically.
type Scene struct {
	ID        uint64        `json:"id"`
	UserID    uint64        `json:"user_id"`
	Name      string        `json:"name"`
	Icon      *string       `json:"icon,omitempty"`
	Actions   []SceneAction `json:"actions,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// SceneAction represents a single device command within a scene.
type SceneAction struct {
	ID        uint64    `json:"id"`
	SceneID   uint64    `json:"scene_id"`
	DeviceID  uint64    `json:"device_id"`
	Command   string    `json:"command"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// SceneActionResult is the outcome of executing a single SceneAction.
type SceneActionResult struct {
	DeviceID uint64`json:"device_id"`
	Command  string `json:"command"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// SceneExecutionResult summarises the result of executing a Scene.
type SceneExecutionResult struct {
	SceneID uint64              `json:"scene_id"`
	Actions []SceneActionResult `json:"actions"`
}
