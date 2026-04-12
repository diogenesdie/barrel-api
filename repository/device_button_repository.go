package repository

import (
	"barrel-api/model"
	"database/sql"
)

type DeviceButtonRepository struct {
	db *sql.DB
}

func NewDeviceButtonRepository(db *sql.DB) *DeviceButtonRepository {
	return &DeviceButtonRepository{db}
}

func (r *DeviceButtonRepository) GetButtonsByDeviceID(deviceID uint64) ([]model.DeviceButton, error) {
	rows, err := r.db.Query(`
		SELECT id, device_id, original_name, protocol, address, command, label, created_at, updated_at
		  FROM barrel.device_buttons
		 WHERE device_id = $1
		 ORDER BY id
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buttons := []model.DeviceButton{}
	for rows.Next() {
		var b model.DeviceButton
		if err := rows.Scan(
			&b.ID, &b.DeviceID, &b.OriginalName, &b.Protocol,
			&b.Address, &b.Command, &b.Label, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		buttons = append(buttons, b)
	}
	return buttons, nil
}

// UpsertButtons inserts or updates buttons for a device.
// Uses ON CONFLICT (device_id, original_name) DO UPDATE so the app can sync repeatedly.
func (r *DeviceButtonRepository) UpsertButtons(deviceID uint64, buttons []model.DeviceButton) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO barrel.device_buttons (device_id, original_name, protocol, address, command, label)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (device_id, original_name) DO UPDATE
		  SET protocol     = EXCLUDED.protocol,
		      address      = EXCLUDED.address,
		      command      = EXCLUDED.command,
		      label        = EXCLUDED.label,
		      updated_at   = now()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range buttons {
		if _, err = stmt.Exec(deviceID, b.OriginalName, b.Protocol, b.Address, b.Command, b.Label); err != nil {
			return err
		}
	}

	return tx.Commit()
}
