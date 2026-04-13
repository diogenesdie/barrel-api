package repository

import (
	"barrel-api/model"
	"database/sql"
	"errors"
)

var ErrSceneNotFound = errors.New("scene not found")

// SceneRepositoryInterface defines the contract for scene persistence.
type SceneRepositoryInterface interface {
	CreateScene(scene *model.Scene) (uint64, error)
	GetSceneByID(id uint64) (*model.Scene, error)
	GetScenesByUser(userID uint64) ([]model.Scene, error)
	UpdateScene(scene *model.Scene) error
	DeleteScene(id uint64, userID uint64) error
}

// SceneRepository implements SceneRepositoryInterface using PostgreSQL.
type SceneRepository struct {
	db *sql.DB
}

func NewSceneRepository(db *sql.DB) *SceneRepository {
	return &SceneRepository{db: db}
}

// CreateScene persists a new scene and its actions in a single transaction.
func (r *SceneRepository) CreateScene(scene *model.Scene) (uint64, error) {
	tx, err := r.db.Begin()
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
		INSERT INTO barrel.scenes (user_id, name, icon)
		VALUES ($1, $2, $3)
		RETURNING id
	`, scene.UserID, scene.Name, scene.Icon).Scan(&id)
	if err != nil {
		return 0, err
	}

	if len(scene.Actions) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO barrel.scene_actions (scene_id, device_id, command, sort_order)
			VALUES ($1, $2, $3, $4)
		`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()

		for _, a := range scene.Actions {
			if _, err = stmt.Exec(id, a.DeviceID, a.Command, a.SortOrder); err != nil {
				return 0, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// GetSceneByID returns a scene with its actions or ErrSceneNotFound.
func (r *SceneRepository) GetSceneByID(id uint64) (*model.Scene, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, name, icon, created_at, updated_at
		FROM barrel.scenes
		WHERE id = $1 AND deleted_at IS NULL
	`, id)

	s := &model.Scene{}
	err := row.Scan(&s.ID, &s.UserID, &s.Name, &s.Icon, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrSceneNotFound
	}
	if err != nil {
		return nil, err
	}

	s.Actions, err = r.getActionsBySceneID(id)
	return s, err
}

// GetScenesByUser returns all active scenes for a user, including their actions.
func (r *SceneRepository) GetScenesByUser(userID uint64) ([]model.Scene, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, name, icon, created_at, updated_at
		FROM barrel.scenes
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scenes := []model.Scene{}
	for rows.Next() {
		var s model.Scene
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Icon, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		scenes = append(scenes, s)
	}

	for i, s := range scenes {
		actions, err := r.getActionsBySceneID(s.ID)
		if err != nil {
			return nil, err
		}
		scenes[i].Actions = actions
	}

	return scenes, nil
}

// UpdateScene replaces a scene's metadata and actions atomically.
func (r *SceneRepository) UpdateScene(scene *model.Scene) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	res, err := tx.Exec(`
		UPDATE barrel.scenes
		SET name = $1, icon = $2, updated_at = now()
		WHERE id = $3 AND user_id = $4 AND deleted_at IS NULL
	`, scene.Name, scene.Icon, scene.ID, scene.UserID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		tx.Rollback()
		return ErrSceneNotFound
	}

	if _, err = tx.Exec(`DELETE FROM barrel.scene_actions WHERE scene_id = $1`, scene.ID); err != nil {
		return err
	}

	if len(scene.Actions) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO barrel.scene_actions (scene_id, device_id, command, sort_order)
			VALUES ($1, $2, $3, $4)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, a := range scene.Actions {
			if _, err = stmt.Exec(scene.ID, a.DeviceID, a.Command, a.SortOrder); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// DeleteScene soft-deletes a scene, verifying ownership.
func (r *SceneRepository) DeleteScene(id uint64, userID uint64) error {
	res, err := r.db.Exec(`
		UPDATE barrel.scenes
		SET deleted_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, id, userID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrSceneNotFound
	}
	return nil
}

func (r *SceneRepository) getActionsBySceneID(sceneID uint64) ([]model.SceneAction, error) {
	rows, err := r.db.Query(`
		SELECT id, scene_id, device_id, command, sort_order, created_at
		FROM barrel.scene_actions
		WHERE scene_id = $1
		ORDER BY sort_order ASC
	`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []model.SceneAction{}
	for rows.Next() {
		var a model.SceneAction
		if err := rows.Scan(&a.ID, &a.SceneID, &a.DeviceID, &a.Command, &a.SortOrder, &a.CreatedAt); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, nil
}
