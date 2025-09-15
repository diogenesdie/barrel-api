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
	_, err := gr.db.Exec(`
		insert into barrel.groups (user_id, name, position, icon, is_default)
		values ($1, $2, $3, $4, $5)
	`, group.UserID, group.Name, group.Position, group.Icon, group.IsDefault)

	return err
}

func (gr *GroupRepository) GetGroupByID(id uint64) (*model.Group, error) {
	row := gr.db.QueryRow(`
		select g.id
		      ,g.user_id
			  ,g.name
			  ,g.icon
			  ,g.position
			  ,g.is_default
			  ,g.created_at
			  ,g.updated_at
		  from barrel.groups g
		 where g.id = $1
	`, id)

	group := &model.Group{}

	err := row.Scan(&group.ID, &group.UserID, &group.Name, &group.Icon, &group.Position, &group.IsDefault, &group.CreatedAt, &group.UpdatedAt)
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
		if err := rows.Scan(&group.ID, &group.UserID, &group.Name, &group.Icon, &group.Position, &group.IsDefault, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (gr *GroupRepository) UpdateGroup(group *model.Group) error {
	res, err := gr.db.Exec(`
		update barrel.groups
		   set name       = $1
		      ,position   = $2
			  ,icon       = $3
			  ,is_default = $4
		      ,updated_at = now()
		 where id         = $5
	`, group.Name, group.Position, group.Icon, group.IsDefault, group.ID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrGroupNotFound
	}

	return nil
}

func (gr *GroupRepository) DeleteGroup(id uint64) error {
	res, err := gr.db.Exec(`
		delete from barrel.groups
		 where id         = $1
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
