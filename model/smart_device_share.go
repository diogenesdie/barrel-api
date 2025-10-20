package model

import (
	"time"
)

type SmartDeviceShare struct {
	ID            uint64     `db:"id" json:"id"`
	DeviceShareID uint64     `db:"device_share_id" json:"device_share_id"`
	DeviceID      uint64     `db:"device_id" json:"device_id"`
	UserID        uint64     `db:"user_id" json:"user_id"`
	GroupID       *uint64    `db:"group_id" json:"group_id"`
	IsFavorite    bool       `db:"is_favorite" json:"is_favorite"`
	Name          *string    `db:"name" json:"name"`
	Icon          *string    `db:"icon" json:"icon"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at" json:"deleted_at"`
}
