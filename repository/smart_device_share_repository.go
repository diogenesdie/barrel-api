package repository

import (
	"barrel-api/model"
	"database/sql"
)

var ErrSmartDeviceShareNotFound = sql.ErrNoRows

type SmartDeviceShareRepository struct {
	DB *sql.DB
}

func NewSmartDeviceShareRepository(db *sql.DB) *SmartDeviceShareRepository {
	return &SmartDeviceShareRepository{DB: db}
}

func (r *SmartDeviceShareRepository) GetSmartDeviceShareByDeviceShareID(deviceShareID uint64) (*model.SmartDeviceShare, error) {
	query := `
		SELECT id, device_share_id, group_id, is_favorite, name, icon,
		       created_at, updated_at, deleted_at
		  FROM barrel.smart_devices_share
		 WHERE device_share_id = $1
		   AND deleted_at IS NULL
	`
	var sds model.SmartDeviceShare
	err := r.DB.QueryRow(query, deviceShareID).Scan(
		&sds.ID,
		&sds.DeviceShareID,
		&sds.GroupID,
		&sds.IsFavorite,
		&sds.Name,
		&sds.Icon,
		&sds.CreatedAt,
		&sds.UpdatedAt,
		&sds.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sds, nil
}

func (r *SmartDeviceShareRepository) CreateSmartDeviceShare(sds *model.SmartDeviceShare) error {
	query := `
		INSERT INTO barrel.smart_devices_share
		       (device_share_id, group_id, is_favorite, name, icon)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.DB.QueryRow(
		query,
		sds.DeviceShareID,
		sds.GroupID,
		sds.IsFavorite,
		sds.Name,
		sds.Icon,
	).Scan(&sds.ID, &sds.CreatedAt, &sds.UpdatedAt)
}

func (r *SmartDeviceShareRepository) UpsertSmartDeviceShare(sds *model.SmartDeviceShare) error {
	query := `
		INSERT INTO barrel.smart_devices_share 
		       (device_share_id, device_id, user_id, group_id, is_favorite, name, icon)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, user_id)
		DO UPDATE SET group_id    = EXCLUDED.group_id,
		              is_favorite = EXCLUDED.is_favorite,
		              name        = EXCLUDED.name,
		              icon        = EXCLUDED.icon,
		              updated_at  = now()
		RETURNING id, created_at, updated_at
	`
	return r.DB.QueryRow(
		query,
		sds.DeviceShareID,
		sds.DeviceID,
		sds.UserID,
		sds.GroupID,
		sds.IsFavorite,
		sds.Name,
		sds.Icon,
	).Scan(&sds.ID, &sds.CreatedAt, &sds.UpdatedAt)
}
