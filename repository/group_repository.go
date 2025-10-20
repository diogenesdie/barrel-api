package repository

import (
	"barrel-api/model"
	"database/sql"
	"errors"
)

type GroupRepository struct {
	db *sql.DB
}

var ErrGroupNotFound = errors.New("group not found")

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db}
}

func (gr *GroupRepository) CreateGroup(group *model.Group) error {
	tx, err := gr.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if group.IsDefault {
		if _, err := tx.Exec(`
			update barrel.groups
			   set is_default = false
			 where user_id = $1
			   and is_default = true
		`, group.UserID); err != nil {
			return err
		}
	}

	err = tx.QueryRow(`
		insert into barrel.groups (user_id, name, position, icon, is_default, is_share_group)
		values ($1, $2, $3, $4, $5, $6)
		returning id
	`, group.UserID, group.Name, group.Position, group.Icon, group.IsDefault, group.IsShareGroup).Scan(&group.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (gr *GroupRepository) GetGroupByID(id uint64) (*model.Group, error) {
	row := gr.db.QueryRow(`
		select g.id
		      ,g.user_id
			  ,g.name
			  ,g.icon
			  ,g.position
			  ,g.is_default
			  ,g.is_share_group
			  ,g.created_at
			  ,g.updated_at
		  from barrel.groups g
		 where g.id = $1
	`, id)

	group := &model.Group{}

	err := row.Scan(&group.ID, &group.UserID, &group.Name, &group.Icon, &group.Position, &group.IsDefault, &group.IsShareGroup, &group.CreatedAt, &group.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrGroupNotFound
	}

	return group, err
}

func (gr *GroupRepository) GetGroupsByUser(userID uint64) ([]model.Group, error) {
	rows, err := gr.db.Query(`
		select g.id
		      ,g.user_id
			  ,g.name
			  ,g.icon
			  ,g.position
			  ,g.is_default
			  ,g.is_share_group
			  ,g.created_at
			  ,g.updated_at
		  from barrel.groups g
		 where g.user_id = $1
		 order by g.position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []model.Group{}
	for rows.Next() {
		group := model.Group{}
		if err := rows.Scan(&group.ID, &group.UserID, &group.Name, &group.Icon, &group.Position, &group.IsDefault, &group.IsShareGroup, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (gr *GroupRepository) UpdateGroup(group *model.Group) error {
	tx, err := gr.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if group.IsDefault {
		// desmarca todos os outros do usuário
		if _, err := tx.Exec(`
			update barrel.groups
			   set is_default = false
			 where user_id = $1
			   and id <> $2
		`, group.UserID, group.ID); err != nil {
			return err
		}
	}

	res, err := tx.Exec(`
		update barrel.groups
		   set name       = $1,
		       position   = $2,
		       icon       = $3,
		       is_default = $4,
		       updated_at = now()
		 where id = $5
	`, group.Name, group.Position, group.Icon, group.IsDefault, group.ID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrGroupNotFound
	}

	return tx.Commit()
}

func (gr *GroupRepository) DeleteGroup(id uint64) error {
	group, err := gr.GetGroupByID(id)
	if err != nil {
		return err
	}
	if group.IsDefault {
		return errors.New("cannot delete default group")
	}

	res, err := gr.db.Exec(`
		delete from barrel.groups
		 where id = $1
	`, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrGroupNotFound
	}

	return nil
}
