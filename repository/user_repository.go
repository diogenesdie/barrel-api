package repository

import (
	"barrel-api/model"
	"database/sql"
	"errors"
)

type UserRepository struct {
	db *sql.DB
}

var ErrUserNotFound = errors.New("user not found")

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db}
}

func (ur *UserRepository) CreateUser(user *model.User) (uint64, error) {
	var id uint64

	err := ur.db.QueryRow(`
		INSERT INTO barrel.users (
			id,
			type,
			username,
			name,
			email,
			password,
			plan_id
		) VALUES (
			nextval('barrel.seq_users'),
			$1, $2, $3, $4, $5, 1
		)
		RETURNING id
	`, user.Type, user.Username, user.Name, user.Email, user.Password).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (ur *UserRepository) GetUserByID(id uint64) (*model.User, error) {
	row := ur.db.QueryRow(`
		select u.id
		      ,u.username
			  ,u.name
			  ,u.type
			  ,u.email
	          ,u.created_at
			  ,u.updated_at
		 from barrel.users u 
		where u.id = $1::bigint
		  and u.status = 'A'
	`, id)

	user := &model.User{}

	err := row.Scan(&user.ID, &user.Username, &user.Name, &user.Type, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	return user, err
}

func (ur *UserRepository) GetUsers() ([]model.User, error) {
	rows, err := ur.db.Query(`
		select u.id
		      ,u.username
			  ,u.name
			  ,u.type
			  ,u.email
	          ,u.created_at
			  ,u.updated_at
		 from barrel.users u
		where u.status = 'A'
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []model.User{}

	for rows.Next() {
		user := model.User{}
		err := rows.Scan(&user.ID, &user.Username, &user.Name, &user.Type, &user.Email, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (ur *UserRepository) UpdateUser(user *model.User) error {
	_, err := ur.db.Exec(`
		update barrel.users
		   set type       = $1
		      ,username   = $2
			  ,name       = $3
			  ,email      = $4
			  ,updated_at = now()
		 where id         = $5
	`, user.Type, user.Username, user.Name, user.Email, user.ID)

	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}

	return err
}

func (ur *UserRepository) DeleteUser(id uint64) error {
	_, err := ur.db.Exec(`
		update barrel.users
		   set status     = 'D'
		      ,deleted_at = now()
		 where id         = $1
	`, id)

	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}

	return err
}

func (ur *UserRepository) GetUserIDByCode(code string) (uint64, error) {
	var id uint64

	err := ur.db.QueryRow(`
		select u.id
		  from barrel.users u
		 where u.code = $1
		   and u.status = 'A'
	`, code).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, ErrUserNotFound
	}

	return id, err
}
