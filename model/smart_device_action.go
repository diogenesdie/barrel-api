package model

import "time"

type SmartDeviceAction struct {
	ID                uint64    `json:"id"`
	TriggerDeviceID   uint64    `json:"trigger_device_id"`
	TriggerEvent      string    `json:"trigger_event"`
	TargetDeviceID    uint64    `json:"target_device_id"`
	ActionType        string    `json:"action_type"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	TargetDeviceName  string    `json:"target_device_name,omitempty"`
	TargetDeviceIP    *string   `json:"target_device_ip,omitempty"`
	TargetDeviceQueue string    `json:"target_device_queue,omitempty"`
}
