package model

import "time"

type SmartDevice struct {
	ID                uint64    `json:"id"`
	UserID            uint64    `json:"user_id"`
	GroupID           *uint64   `json:"group_id,omitempty"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	IP                *string   `json:"ip,omitempty"`
	Icon              *string   `json:"icon,omitempty"`
	IVKey             *string   `json:"iv_key,omitempty"`
	State             string    `json:"state"`
	IsFavorite        bool      `json:"is_favorite"`
	IsShared          bool      `json:"is_shared"`
	SSID              *string   `json:"ssid,omitempty"`
	CommunicationMode string    `json:"communication_mode"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
