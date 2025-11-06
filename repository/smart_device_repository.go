package repository

import (
	"barrel-api/model"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

type SmartDeviceRepository struct {
	db *sql.DB
}

var ErrSmartDeviceNotFound = errors.New("smart device not found")

func NewSmartDeviceRepository(db *sql.DB) *SmartDeviceRepository {
	return &SmartDeviceRepository{db}
}

func (sr *SmartDeviceRepository) CreateSmartDevice(device *model.SmartDevice) (uint64, error) {
	tx, err := sr.db.Begin()
	if err != nil {
		return 0, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var id uint64
	err = tx.QueryRow(`
		INSERT INTO barrel.smart_devices (
			user_id, group_id, name, type, ip, iv_key, state,
			is_favorite, ssid, communication_mode, icon, device_id
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id
	`,
		device.UserID, device.GroupID, device.Name, device.Type, device.IP, device.IVKey,
		device.State, device.IsFavorite, device.SSID, device.CommunicationMode, device.Icon,
		device.DeviceID,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	if device.Type == "trigger" && len(device.Actions) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO barrel.smart_device_actions (
				trigger_device_id, trigger_event, target_device_id, action_type
			)
			VALUES ($1, $2, $3, $4)
		`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()

		for _, action := range device.Actions {
			_, err = stmt.Exec(id, action.TriggerEvent, action.TargetDeviceID, action.ActionType)
			if err != nil {
				return 0, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func (sr *SmartDeviceRepository) GetSmartDeviceByID(id uint64) (*model.SmartDevice, error) {
	row := sr.db.QueryRow(`
		select d.id
		      ,d.user_id
			  ,d.group_id
			  ,d.name
			  ,d.type
			  ,d.ip
			  ,d.iv_key
			  ,d.state
			  ,d.is_favorite
			  ,d.ssid
			  ,d.communication_mode
			  ,d.icon
			  ,d.device_id
			  ,d.created_at
			  ,d.updated_at
			  ,u.username as owner_username
		  from barrel.smart_devices d
		      ,barrel.users u
		 where u.id = d.user_id
		   and d.id = $1
		   and d.deleted_at is null
	`, id)

	device := &model.SmartDevice{}
	err := row.Scan(
		&device.ID, &device.UserID, &device.GroupID, &device.Name, &device.Type, &device.IP,
		&device.IVKey, &device.State, &device.IsFavorite, &device.SSID, &device.CommunicationMode,
		&device.Icon, &device.DeviceID, &device.CreatedAt, &device.UpdatedAt, &device.OwnerUsername,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSmartDeviceNotFound
	}

	//Get actions
	actionRows, err := sr.db.Query(`
		select 
			a.id,
			a.trigger_device_id,
			a.trigger_event,
			a.target_device_id,
			t.name as target_device_name,
			t.ip as target_device_ip,
			concat('users/', u.username, '/', t.device_id, '/command') as target_device_queue,
			a.action_type,
			a.created_at,
			a.updated_at
		from barrel.smart_device_actions a
		join barrel.smart_devices t on t.id = a.target_device_id
		join barrel.users u on u.id = t.user_id
		where a.trigger_device_id = $1
		order by a.id;
	`, id)
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()

	actions := []model.SmartDeviceAction{}

	for actionRows.Next() {
		var act model.SmartDeviceAction
		if err := actionRows.Scan(
			&act.ID, &act.TriggerDeviceID, &act.TriggerEvent,
			&act.TargetDeviceID, &act.TargetDeviceName, &act.TargetDeviceIP, &act.TargetDeviceQueue,
			&act.ActionType, &act.CreatedAt, &act.UpdatedAt,
		); err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}

	device.Actions = actions

	return device, err
}

func (sr *SmartDeviceRepository) GetSmartDevicesByUser(userID uint64) ([]model.SmartDevice, error) {
	query := `
		with own_devices as (
			select id, false as is_shared
			from barrel.smart_devices
			where user_id = $1
			and deleted_at is null
		),
		shared_devices as (
			select distinct sd.id, true as is_shared, ds.id as device_share_id
			from barrel.device_shares ds
			join barrel.smart_devices sd 
				on (
					(ds.device_id is not null and sd.id = ds.device_id)
				or (ds.group_id is not null and sd.group_id = ds.group_id)
				)
			where ds.shared_with_id = $1
			and ds.status = 'A'
			and ds.deleted_at is null
			and sd.deleted_at is null
		),
		all_devices as (
			select id, is_shared, null::bigint as device_share_id
			from own_devices
			union
			select id, is_shared, device_share_id
			from shared_devices
		)
		select d.id,
			d.user_id,
			coalesce(sds.group_id, d.group_id) as group_id,
			coalesce(sds.name, d.name)         as name,
			d.type,
			d.ip,
			d.iv_key,
			d.state,
			coalesce(sds.is_favorite, d.is_favorite) as is_favorite,
			d.ssid,
			d.communication_mode,
			coalesce(sds.icon, d.icon)         as icon,
			d.device_id,
			d.created_at,
			d.updated_at,
			ad.is_shared,
			u.username as owner_username
		from barrel.smart_devices d
		join barrel.users u on u.id = d.user_id
		join all_devices ad on ad.id = d.id
		left join barrel.smart_devices_share sds 
				on sds.device_share_id = ad.device_share_id
				and sds.deleted_at is null
		order by d.created_at desc;
	`

	rows, err := sr.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := []model.SmartDevice{}
	deviceIDs := []uint64{}

	for rows.Next() {
		var device model.SmartDevice
		if err := rows.Scan(
			&device.ID, &device.UserID, &device.GroupID, &device.Name, &device.Type, &device.IP,
			&device.IVKey, &device.State, &device.IsFavorite, &device.SSID, &device.CommunicationMode,
			&device.Icon, &device.DeviceID, &device.CreatedAt, &device.UpdatedAt, &device.IsShared, &device.OwnerUsername,
		); err != nil {
			return nil, err
		}

		deviceIDs = append(deviceIDs, device.ID)
		devices = append(devices, device)
	}

	if len(devices) == 0 {
		return devices, nil
	}

	actionQuery := `
		select 
			a.id,
			a.trigger_device_id,
			a.trigger_event,
			a.target_device_id,
			t.name as target_device_name,
			t.ip as target_device_ip,
			concat('users/', u.username, '/', t.device_id, '/command') as target_device_queue,
			a.action_type,
			a.created_at,
			a.updated_at
		from barrel.smart_device_actions a
		join barrel.smart_devices t on t.id = a.target_device_id
		join barrel.users u on u.id = t.user_id
		where a.trigger_device_id = any($1)
		order by a.trigger_device_id, a.id;
	`

	actionRows, err := sr.db.Query(actionQuery, pq.Array(deviceIDs))
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()

	actionMap := make(map[uint64][]model.SmartDeviceAction)

	for actionRows.Next() {
		var act model.SmartDeviceAction
		if err := actionRows.Scan(
			&act.ID, &act.TriggerDeviceID, &act.TriggerEvent,
			&act.TargetDeviceID, &act.TargetDeviceName, &act.TargetDeviceIP, &act.TargetDeviceQueue,
			&act.ActionType, &act.CreatedAt, &act.UpdatedAt,
		); err != nil {
			return nil, err
		}
		actionMap[act.TriggerDeviceID] = append(actionMap[act.TriggerDeviceID], act)
	}

	for i, d := range devices {
		if acts, ok := actionMap[d.ID]; ok {
			devices[i].Actions = acts
		}
	}

	return devices, nil
}

func (sr *SmartDeviceRepository) UpdateSmartDevice(device *model.SmartDevice) error {
	tx, err := sr.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	res, err := tx.Exec(`
		UPDATE barrel.smart_devices
		   SET group_id            = $1,
		       name                = $2,
		       type                = $3,
		       ip                  = $4,
		       iv_key              = $5,
		       state               = $6,
		       is_favorite         = $7,
		       ssid                = $8,
		       communication_mode  = $9,
		       icon                = $10,
		       updated_at          = now()
		 WHERE id                  = $11
		   AND deleted_at IS NULL
	`,
		device.GroupID, device.Name, device.Type, device.IP, device.IVKey, device.State,
		device.IsFavorite, device.SSID, device.CommunicationMode, device.Icon, device.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		return ErrSmartDeviceNotFound
	}

	if device.Type == "trigger" {
		_, err = tx.Exec(`
			DELETE FROM barrel.smart_device_actions
			 WHERE trigger_device_id = $1
		`, device.ID)
		if err != nil {
			tx.Rollback()
			return err
		}

		if len(device.Actions) > 0 {
			stmt, err := tx.Prepare(`
				INSERT INTO barrel.smart_device_actions (
					trigger_device_id, trigger_event, target_device_id, action_type
				)
				VALUES ($1, $2, $3, $4)
			`)
			if err != nil {
				tx.Rollback()
				return err
			}
			defer stmt.Close()

			for _, act := range device.Actions {
				_, err = stmt.Exec(device.ID, act.TriggerEvent, act.TargetDeviceID, act.ActionType)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	if device.Type != "trigger" {
		_, _ = tx.Exec(`DELETE FROM barrel.smart_device_actions WHERE trigger_device_id = $1`, device.ID)
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (sr *SmartDeviceRepository) DeleteSmartDevice(id string) error {
	res, err := sr.db.Exec(`
		update barrel.smart_devices
		   set deleted_at = now()
		 where id         = $1
		   and deleted_at is null
	`, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrSmartDeviceNotFound
	}

	return nil
}
