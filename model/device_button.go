package model

import "time"

type DeviceButton struct {
	ID           uint64    `json:"id"`
	DeviceID     uint64    `json:"device_id"`
	OriginalName string    `json:"original_name"`
	Protocol     string    `json:"protocol"`
	Address      int       `json:"address"`
	Command      int       `json:"command"`
	Label        string    `json:"label"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UpsertDeviceButtonsRequest struct {
	Buttons []DeviceButton `json:"buttons"`
}
