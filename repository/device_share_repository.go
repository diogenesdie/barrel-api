package repository

import (
	"barrel-api/model"
	"database/sql"
	"errors"
)

type DeviceShareRepository struct {
	db *sql.DB
}

var (
	ErrDeviceShareNotFound = errors.New("device share not found")
	ErrAlreadyShared       = errors.New("resource is already shared with this user")
)

func NewDeviceShareRepository(db *sql.DB) *DeviceShareRepository {
	return &DeviceShareRepository{db}
}

func (r *DeviceShareRepository) ExistsActiveShare(deviceID *uint64, groupID *uint64, sharedWithID uint64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		select count(*) 
		  from barrel.device_shares 
		 where shared_with_id = $1
		   and status in ('P','A')
		   and (
			    (device_id = $2 and $2 is not null) or
			    (group_id = $3 and $3 is not null)
		   )
		   and deleted_at is null
	`, sharedWithID, deviceID, groupID).Scan(&count)
	return count > 0, err
}

func (r *DeviceShareRepository) Create(ds *model.DeviceShare) error {
	_, err := r.db.Exec(`
		insert into barrel.device_shares
			(owner_id, shared_with_id, device_id, group_id, status)
		values ($1,$2,$3,$4,'P')
	`, ds.OwnerID, ds.SharedWithID, ds.DeviceID, ds.GroupID)
	return err
}

func (r *DeviceShareRepository) GetByID(id uint64) (*model.DeviceShare, error) {
	row := r.db.QueryRow(`
		select id, owner_id, shared_with_id, device_id, group_id, status,
			   accepted_at, revoked_at, created_at, updated_at
		  from barrel.device_shares
		 where id = $1 and deleted_at is null
	`, id)

	var ds model.DeviceShare
	err := row.Scan(&ds.ID, &ds.OwnerID, &ds.SharedWithID, &ds.DeviceID, &ds.GroupID,
		&ds.Status, &ds.AcceptedAt, &ds.RevokedAt, &ds.CreatedAt, &ds.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDeviceShareNotFound
	}
	return &ds, err
}

func (r *DeviceShareRepository) GetByUser(userID uint64) ([]model.DeviceShare, error) {
	rows, err := r.db.Query(`
		select id, owner_id, shared_with_id, device_id, group_id, status,
			   accepted_at, revoked_at, created_at, updated_at
		  from barrel.device_shares
		 where (owner_id = $1 or shared_with_id = $1) and deleted_at is null
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.DeviceShare
	for rows.Next() {
		var ds model.DeviceShare
		if err := rows.Scan(&ds.ID, &ds.OwnerID, &ds.SharedWithID, &ds.DeviceID, &ds.GroupID,
			&ds.Status, &ds.AcceptedAt, &ds.RevokedAt, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, ds)
	}
	return list, nil
}

func (r *DeviceShareRepository) UpdateStatus(id uint64, status string) error {
	_, err := r.db.Exec(`
		update barrel.device_shares
		   set status = $1,
		       updated_at = now()
		 where id = $2 and deleted_at is null
	`, status, id)
	return err
}
