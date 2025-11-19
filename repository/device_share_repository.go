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
		select ds.id, ds.owner_id, u.name as owner_name, ds.shared_with_id, ds.device_id, ds.group_id, ds.status,
			   ds.accepted_at, ds.revoked_at, ds.created_at, ds.updated_at
		  from barrel.device_shares ds
		      ,barrel.users u
		 where u.id = ds.owner_id
		   and ds.id = $1 
		   and ds.deleted_at is null
	`, id)

	var ds model.DeviceShare
	err := row.Scan(&ds.ID, &ds.OwnerID, &ds.OwnerName, &ds.SharedWithID, &ds.DeviceID, &ds.GroupID,
		&ds.Status, &ds.AcceptedAt, &ds.RevokedAt, &ds.CreatedAt, &ds.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDeviceShareNotFound
	}
	return &ds, err
}

func (r *DeviceShareRepository) GetByUser(userID uint64) ([]model.DeviceShare, error) {
	rows, err := r.db.Query(`
		select ds.id
		      ,ds.owner_id
			  ,u.name as owner_name
			  ,ds.shared_with_id
			  ,su.name as shared_with_name
			  ,ds.device_id
			  ,ds.group_id
			  ,coalesce((select d.name from barrel.smart_devices d where d.id = ds.device_id), (select g.name from barrel.groups g where g.id = ds.group_id)) as shared_item_name
			  ,case when ds.device_id is not null then 'device' when ds.group_id is not null then 'group' else 'unknown' end as type
			  ,ds.status
			  ,ds.accepted_at
			  ,ds.revoked_at
			  ,ds.created_at
			  ,ds.updated_at
		  from barrel.device_shares ds
		      ,barrel.users u
			  ,barrel.users su
		 where u.id = ds.owner_id
		   and su.id = ds.shared_with_id
		   and (ds.owner_id = $1 or ds.shared_with_id = $1) and ds.deleted_at is null
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.DeviceShare
	for rows.Next() {
		var ds model.DeviceShare
		if err := rows.Scan(&ds.ID, &ds.OwnerID, &ds.OwnerName, &ds.SharedWithID, &ds.SharedWithName, &ds.DeviceID, &ds.GroupID, &ds.SharedItemName, &ds.Type,
			&ds.Status, &ds.AcceptedAt, &ds.RevokedAt, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, ds)
	}
	return list, nil
}

func (r *DeviceShareRepository) UpdateStatus(id uint64, status string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Atualiza status e já retorna dados necessários para decidir o que fazer
	var (
		shareID          int64
		sharedWithID     int64
		deviceIDNullable sql.NullInt64
		groupIDNullable  sql.NullInt64
		newStatus        string
	)
	err = tx.QueryRow(`
		update barrel.device_shares ds
		   set status = $1,
		       updated_at = now()
		 where ds.id = $2
		   and ds.deleted_at is null
		returning ds.id, ds.shared_with_id, ds.device_id, ds.group_id, ds.status
	`, status, id).Scan(&shareID, &sharedWithID, &deviceIDNullable, &groupIDNullable, &newStatus)
	if err != nil {
		return err
	}

	// Só cria smart_devices_share quando a share fica ativa
	if newStatus != "A" {
		return tx.Commit()
	}

	// Descobre o group_id "recebedor" do usuário (grupo com is_share_group = true)
	var recvShareGroupID sql.NullInt64
	err = tx.QueryRow(`
		select g.id
		  from barrel.groups g
		 where g.user_id = $1
		   and g.is_share_group = true
		 order by g.id
		 limit 1
	`, sharedWithID).Scan(&recvShareGroupID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	// Se não existir, vamos seguir com NULL (o campo permite null e tem ON DELETE SET NULL)
	stmt, err := tx.Prepare(`
		insert into barrel.smart_devices_share
			(device_share_id, device_id, user_id, group_id, is_favorite)
		values
			($1, $2, $3, $4, false)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Caso 1: compartilhamento de 1 device
	if deviceIDNullable.Valid {
		_, err = stmt.Exec(shareID, deviceIDNullable.Int64, sharedWithID,
			func() any {
				if recvShareGroupID.Valid {
					return recvShareGroupID.Int64
				}
				return nil
			}(),
		)
		if err != nil {
			return err
		}
	} else if groupIDNullable.Valid {
		// Caso 2: compartilhamento de grupo -> criar para todos os devices do grupo compartilhado
		rows, qerr := tx.Query(`
			select d.id
			  from barrel.smart_devices d
			 where d.group_id = $1
			   and d.deleted_at is null
		`, groupIDNullable.Int64)
		if qerr != nil {
			err = qerr
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var devID int64
			if err = rows.Scan(&devID); err != nil {
				return err
			}
			if _, err = stmt.Exec(shareID, devID, sharedWithID,
				func() any {
					if recvShareGroupID.Valid {
						return recvShareGroupID.Int64
					}
					return nil
				}(),
			); err != nil {
				return err
			}
		}
		if err = rows.Err(); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *DeviceShareRepository) GetActiveShareByDeviceAndUser(deviceID uint64, userID uint64) (*model.DeviceShare, error) {
	row := r.db.QueryRow(`
		select ds.id, ds.owner_id, u.name as owner_name, ds.shared_with_id, ds.device_id, ds.group_id, ds.status,
			   ds.accepted_at, ds.revoked_at, ds.created_at, ds.updated_at
		  from barrel.device_shares ds
		      ,barrel.users u
		 where u.id = ds.owner_id
		   and ds.device_id = $1 
		   and ds.shared_with_id = $2
		   and ds.status = 'A'
		   and ds.deleted_at is null
	`, deviceID, userID)

	var ds model.DeviceShare
	err := row.Scan(&ds.ID, &ds.OwnerID, &ds.OwnerName, &ds.SharedWithID, &ds.DeviceID, &ds.GroupID,
		&ds.Status, &ds.AcceptedAt, &ds.RevokedAt, &ds.CreatedAt, &ds.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDeviceShareNotFound
	}
	return &ds, err
}
