package model

import "time"

type DeviceShare struct {
	ID           uint64     `json:"id"`
	OwnerID      uint64     `json:"owner_id"`
	SharedWithID uint64     `json:"shared_with_id"`
	DeviceID     *uint64    `json:"device_id,omitempty"`
	GroupID      *uint64    `json:"group_id,omitempty"`
	Status       string     `json:"status"` // P, A, R
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
