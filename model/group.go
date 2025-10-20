package model

import "time"

type Group struct {
	ID           uint64    `json:"id"`
	UserID       uint64    `json:"user_id"`
	Name         string    `json:"name"`
	Icon         *string   `json:"icon,omitempty"`
	IsDefault    bool      `json:"is_default"`
	IsShareGroup bool      `json:"is_share_group"`
	Position     int       `json:"position"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
