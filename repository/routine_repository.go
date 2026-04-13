package repository

import (
	"barrel-api/model"
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrRoutineNotFound = errors.New("routine not found")

// RoutineRepositoryInterface defines the contract for routine persistence.
type RoutineRepositoryInterface interface {
	CreateRoutine(r *model.Routine) (uint64, error)
	GetRoutineByID(id uint64) (*model.Routine, error)
	GetRoutinesByUser(userID uint64) ([]model.Routine, error)
	GetEnabledRoutinesByTriggerType(triggerType string) ([]model.Routine, error)
	UpdateRoutine(r *model.Routine) error
	DeleteRoutine(id uint64, userID uint64) error
}

// RoutineRepository implements RoutineRepositoryInterface using PostgreSQL.
type RoutineRepository struct {
	db *sql.DB
}

func NewRoutineRepository(db *sql.DB) *RoutineRepository {
	return &RoutineRepository{db: db}
}

// CreateRoutine persists a new routine and its actions in a single transaction.
func (r *RoutineRepository) CreateRoutine(routine *model.Routine) (uint64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	expectedStateJSON, err := marshalExpectedState(routine.Trigger.ExpectedState)
	if err != nil {
		return 0, err
	}

	var id uint64
	err = tx.QueryRow(`
		INSERT INTO barrel.routines
			(user_id, name, enabled, trigger_type, trigger_device_id, trigger_expected_state, trigger_cron)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, routine.UserID, routine.Name, routine.Enabled,
		routine.Trigger.Type, routine.Trigger.DeviceID, expectedStateJSON, routine.Trigger.Cron,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if err = r.insertActions(tx, id, routine.Actions); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// GetRoutineByID returns a routine with its actions or ErrRoutineNotFound.
func (r *RoutineRepository) GetRoutineByID(id uint64) (*model.Routine, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, name, enabled,
		       trigger_type, trigger_device_id, trigger_expected_state, trigger_cron,
		       created_at, updated_at
		FROM barrel.routines
		WHERE id = $1 AND deleted_at IS NULL
	`, id)

	routine, err := scanRoutine(row)
	if err == sql.ErrNoRows {
		return nil, ErrRoutineNotFound
	}
	if err != nil {
		return nil, err
	}

	routine.Actions, err = r.getActionsByRoutineID(id)
	return routine, err
}

// GetRoutinesByUser returns all active routines for a user.
func (r *RoutineRepository) GetRoutinesByUser(userID uint64) ([]model.Routine, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, name, enabled,
		       trigger_type, trigger_device_id, trigger_expected_state, trigger_cron,
		       created_at, updated_at
		FROM barrel.routines
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRoutines(rows)
}

// GetEnabledRoutinesByTriggerType returns all enabled routines of a given trigger type.
// Used by the scheduler (type="schedule") and MQTT listener (type="device").
func (r *RoutineRepository) GetEnabledRoutinesByTriggerType(triggerType string) ([]model.Routine, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, name, enabled,
		       trigger_type, trigger_device_id, trigger_expected_state, trigger_cron,
		       created_at, updated_at
		FROM barrel.routines
		WHERE trigger_type = $1 AND enabled = true AND deleted_at IS NULL
	`, triggerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRoutines(rows)
}

// UpdateRoutine replaces a routine's data and actions atomically.
func (r *RoutineRepository) UpdateRoutine(routine *model.Routine) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	expectedStateJSON, err := marshalExpectedState(routine.Trigger.ExpectedState)
	if err != nil {
		return err
	}

	res, err := tx.Exec(`
		UPDATE barrel.routines
		SET name = $1, enabled = $2,
		    trigger_type = $3, trigger_device_id = $4,
		    trigger_expected_state = $5, trigger_cron = $6,
		    updated_at = now()
		WHERE id = $7 AND user_id = $8 AND deleted_at IS NULL
	`, routine.Name, routine.Enabled,
		routine.Trigger.Type, routine.Trigger.DeviceID,
		expectedStateJSON, routine.Trigger.Cron,
		routine.ID, routine.UserID,
	)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		tx.Rollback()
		return ErrRoutineNotFound
	}

	if _, err = tx.Exec(`DELETE FROM barrel.routine_actions WHERE routine_id = $1`, routine.ID); err != nil {
		return err
	}

	if err = r.insertActions(tx, routine.ID, routine.Actions); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteRoutine soft-deletes a routine, verifying ownership.
func (r *RoutineRepository) DeleteRoutine(id uint64, userID uint64) error {
	res, err := r.db.Exec(`
		UPDATE barrel.routines
		SET deleted_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, id, userID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrRoutineNotFound
	}
	return nil
}

// --- helpers ---

func (r *RoutineRepository) insertActions(tx *sql.Tx, routineID uint64, actions []model.RoutineAction) error {
	if len(actions) == 0 {
		return nil
	}
	stmt, err := tx.Prepare(`
		INSERT INTO barrel.routine_actions (routine_id, action_type, device_id, command, scene_id, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, a := range actions {
		if _, err = stmt.Exec(routineID, a.Type, a.DeviceID, a.Command, a.SceneID, a.SortOrder); err != nil {
			return err
		}
	}
	return nil
}

func (r *RoutineRepository) getActionsByRoutineID(routineID uint64) ([]model.RoutineAction, error) {
	rows, err := r.db.Query(`
		SELECT id, routine_id, action_type, device_id, command, scene_id, sort_order, created_at
		FROM barrel.routine_actions
		WHERE routine_id = $1
		ORDER BY sort_order ASC
	`, routineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []model.RoutineAction{}
	for rows.Next() {
		var a model.RoutineAction
		if err := rows.Scan(&a.ID, &a.RoutineID, &a.Type, &a.DeviceID, &a.Command, &a.SceneID, &a.SortOrder, &a.CreatedAt); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, nil
}

func (r *RoutineRepository) scanRoutines(rows *sql.Rows) ([]model.Routine, error) {
	routines := []model.Routine{}
	for rows.Next() {
		routine, err := scanRoutineRow(rows)
		if err != nil {
			return nil, err
		}
		routines = append(routines, *routine)
	}
	for i, rt := range routines {
		actions, err := r.getActionsByRoutineID(rt.ID)
		if err != nil {
			return nil, err
		}
		routines[i].Actions = actions
	}
	return routines, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRoutine(row scannable) (*model.Routine, error) {
	var rt model.Routine
	var expectedStateJSON []byte
	err := row.Scan(
		&rt.ID, &rt.UserID, &rt.Name, &rt.Enabled,
		&rt.Trigger.Type, &rt.Trigger.DeviceID, &expectedStateJSON, &rt.Trigger.Cron,
		&rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(expectedStateJSON) > 0 {
		if err := json.Unmarshal(expectedStateJSON, &rt.Trigger.ExpectedState); err != nil {
			return nil, err
		}
	}
	return &rt, nil
}

func scanRoutineRow(rows *sql.Rows) (*model.Routine, error) {
	var rt model.Routine
	var expectedStateJSON []byte
	err := rows.Scan(
		&rt.ID, &rt.UserID, &rt.Name, &rt.Enabled,
		&rt.Trigger.Type, &rt.Trigger.DeviceID, &expectedStateJSON, &rt.Trigger.Cron,
		&rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(expectedStateJSON) > 0 {
		if err := json.Unmarshal(expectedStateJSON, &rt.Trigger.ExpectedState); err != nil {
			return nil, err
		}
	}
	return &rt, nil
}

func marshalExpectedState(state map[string]string) ([]byte, error) {
	if len(state) == 0 {
		return nil, nil
	}
	return json.Marshal(state)
}
