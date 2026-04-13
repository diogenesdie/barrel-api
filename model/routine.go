package model

import "time"

// RoutineTrigger defines what causes a routine to fire.
// Type "device": fires when a device reaches the expected state (MQTT).
// Type "schedule": fires at the cron-specified time.
type RoutineTrigger struct {
	Type          string            `json:"type"`                     // "device" | "schedule"
	DeviceID      *uint64           `json:"device_id,omitempty"`      // device trigger only
	ExpectedState map[string]string `json:"expected_state,omitempty"` // device trigger only
	Cron          *string           `json:"cron,omitempty"`           // schedule trigger only
}

// RoutineAction is a single step executed when a routine fires.
// Type "device": sends a command to a device.
// Type "scene": activates a scene.
type RoutineAction struct {
	ID        uint64  `json:"id"`
	RoutineID uint64  `json:"routine_id"`
	Type      string  `json:"type"`               // "device" | "scene"
	DeviceID  *uint64 `json:"device_id,omitempty"` // device action only
	Command   *string `json:"command,omitempty"`   // device action only
	SceneID   *uint64 `json:"scene_id,omitempty"`  // scene action only
	SortOrder int     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// Routine represents an automated sequence of actions triggered by an event or schedule.
type Routine struct {
	ID        uint64         `json:"id"`
	UserID    uint64         `json:"user_id"`
	Name      string         `json:"name"`
	Enabled   bool           `json:"enabled"`
	Trigger   RoutineTrigger `json:"trigger"`
	Actions   []RoutineAction `json:"actions,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
