package repository

import (
	"barrel-api/model"
	"barrel-api/token"
	"database/sql"
	"errors"
	"time"
)

type SessionRepository struct {
	db *sql.DB
}

var ErrSessionNotFound = errors.New("session not found")
var ErrSessionExpired = errors.New("session expired")
var ErrSessionInactive = errors.New("session manually inactivated")
var ErrInvalidPassword = errors.New("invalid password")
var ErrUnauthorized = errors.New("unauthorized")
var ErrGenerateToken = errors.New("failed to generate token")
var ErrUpdateToken = errors.New("failed to update token")
var ErrUserAlreadyExists = errors.New("user already exists")

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db}
}

func (sr *SessionRepository) ValidateSession(token string) (*model.Session, error) {
	row := sr.db.QueryRow(`
		select s.id
		      ,s.user_id
			  ,s.token
			  ,s.created_at
			  ,s.updated_at
			  ,s.status
			  ,s.expires_at
		 from barrel.sessions s 
		where s.token = $1::text
	`, token)

	session := &model.Session{}

	err := row.Scan(&session.ID, &session.UserID, &session.Token, &session.CreatedAt, &session.UpdatedAt, &session.Status, &session.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}

	if session.Status == "I" {
		return nil, ErrSessionInactive
	}

	now := time.Now()

	if session.ExpiresAt.Before(now) {
		return nil, ErrSessionExpired
	}

	if err != nil {
		return nil, err
	}

	return session, err
}

func (sr *SessionRepository) Login(login *model.Login) (*model.Session, error) {
	row := sr.db.QueryRow(`
		select u.id
		      ,u.username
		      ,u.name
			  ,u.code
		      ,u.type
		      ,u.email
			  ,u.biometric_login
			  ,u.biometric_edit
			  ,u.biometric_remove
		      ,u.plan_id
		      ,u.created_at
		      ,u.updated_at
		      ,case when crypt($2::text, u.password) = u.password then true else false end as password_match
		  from barrel.users u
		 where u.username = $1::text
		   and u.status = 'A'
	`, login.Username, login.Password)

	var userID uint64
	var username, name, userType, email, code string
	var planID sql.NullInt64
	var userCreatedAt, userUpdatedAt time.Time
	var passwordMatch bool
	var biometricLogin, biometricEdit, biometricRemove bool

	err := row.Scan(&userID, &username, &name, &code, &userType, &email, &biometricLogin, &biometricEdit, &biometricRemove, &planID, &userCreatedAt, &userUpdatedAt, &passwordMatch)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	if !passwordMatch {
		return nil, ErrInvalidPassword
	}

	_, err = sr.db.Exec(`
		update barrel.sessions 
		   set status = 'I',
		       updated_at = current_timestamp
		 where user_id = $1::bigint 
		   and status = 'A'
	`, userID)

	if err != nil {
		return nil, err
	}

	tokenString, err := token.GenerateToken(userID)
	if err != nil {
		return nil, ErrGenerateToken
	}

	now := time.Now()
	session := &model.Session{
		UserID:          userID,
		Username:        username,
		Name:            name,
		Code:            code,
		Email:           email,
		BiometricLogin:  biometricLogin,
		BiometricEdit:   biometricEdit,
		BiometricRemove: biometricRemove,
		Token:           tokenString,
		CreatedAt:       now,
		UpdatedAt:       now,
		Status:          "A",
		ExpiresAt:       now.Add(time.Hour * 24),
	}

	if planID.Valid {
		subscriptionPlan, err := sr.getSubscriptionPlan(planID.Int64)
		if err == nil {
			session.SubscriptionPlan = subscriptionPlan
		}
	}

	err = sr.db.QueryRow(`
		insert into barrel.sessions(id
								   ,user_id
								   ,token
								   ,created_at
								   ,updated_at
								   ,status
								   ,expires_at)
							values (nextval('barrel.seq_sessions')
								   ,$1::bigint
								   ,$2::text
								   ,$3::timestamp
								   ,$4::timestamp
								   ,$5::bpchar
								   ,$6::timestamp)
							returning id
	`, session.UserID, session.Token, session.CreatedAt, session.UpdatedAt, session.Status, session.ExpiresAt).Scan(&session.ID)

	if err != nil {
		return nil, err
	}

	return session, nil
}

func (sr *SessionRepository) Logout(token string) error {
	result, err := sr.db.Exec(`
		update barrel.sessions 
		   set status = 'I',
		       updated_at = current_timestamp
		 where token = $1::text 
		   and status = 'A'
	`, token)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (sr *SessionRepository) LogoutAllSessions(userID uint64) error {
	_, err := sr.db.Exec(`
		update barrel.sessions 
		   set status = 'I',
		       updated_at = current_timestamp
		 where user_id = $1::bigint 
		   and status = 'A'
	`, userID)

	return err
}

// helper para carregar subscription plan
func (sr *SessionRepository) getSubscriptionPlan(planID int64) (*model.SubscriptionPlan, error) {
	row := sr.db.QueryRow(`
		select id
		      ,name
		      ,price
		      ,currency
		      ,max_devices
		      ,local_communication
		      ,mqtt_included
		      ,technical_support
		      ,priority_support
		      ,custom_widgets
		      ,custom_integrations
		      ,is_popular
		      ,is_custom_price
		      ,created_at
		      ,updated_at
		  from barrel.subscription_plans
		 where id = $1::bigint
	`, planID)

	plan := &model.SubscriptionPlan{}
	var price sql.NullFloat64
	var maxDevices sql.NullInt64

	err := row.Scan(
		&plan.ID,
		&plan.Name,
		&price,
		&plan.Currency,
		&maxDevices,
		&plan.LocalCommunication,
		&plan.MQTTIncluded,
		&plan.TechnicalSupport,
		&plan.PrioritySupport,
		&plan.CustomWidgets,
		&plan.CustomIntegrations,
		&plan.IsPopular,
		&plan.IsCustomPrice,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if price.Valid {
		plan.Price = &price.Float64
	}
	if maxDevices.Valid {
		val := int(maxDevices.Int64)
		plan.MaxDevices = &val
	}

	return plan, nil
}

func (sr *SessionRepository) Register(user *model.User) (*model.Session, error) {
	var userID uint64

	err := sr.db.QueryRow(`
		INSERT INTO barrel.users (
			id,
			type,
			username,
			name,
			email,
			password,
			plan_id,
			status,
			created_at,
			updated_at
		) VALUES (
			nextval('barrel.seq_users'),
			$1::text,         -- type
			$2::text,         -- username
			$3::text,         -- name
			$4::text,         -- email
			$5::text,         -- password
			1,                -- plan_id default (ajusta se quiser dinâmico)
			'A',              -- status ativo
			current_timestamp,
			current_timestamp
		)
		RETURNING id
	`, user.Type, user.Username, user.Name, user.Email, user.Password).Scan(&userID)

	if err != nil {
		print(err.Error())
		return nil, err
	}

	var (
		username        string
		name            string
		code            string
		userType        string
		email           string
		biometricLogin  bool
		biometricEdit   bool
		biometricRemove bool
		planID          sql.NullInt64
		userCreatedAt   time.Time
		userUpdatedAt   time.Time
	)

	row := sr.db.QueryRow(`
		select u.username,
		       u.name,
		       u.code,
		       u.type,
		       u.email,
			   u.biometric_login,
			   u.biometric_edit,
			   u.biometric_remove,
		       u.plan_id,
		       u.created_at,
		       u.updated_at
		  from barrel.users u
		 where u.id = $1::bigint
	`, userID)

	err = row.Scan(
		&username,
		&name,
		&code,
		&userType,
		&email,
		&biometricLogin,
		&biometricEdit,
		&biometricRemove,
		&planID,
		&userCreatedAt,
		&userUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_, err = sr.db.Exec(`
		update barrel.sessions 
		   set status = 'I',
		       updated_at = current_timestamp
		 where user_id = $1::bigint 
		   and status = 'A'
	`, userID)

	if err != nil {
		return nil, err
	}

	tokenString, err := token.GenerateToken(userID)
	if err != nil {
		return nil, ErrGenerateToken
	}

	now := time.Now()

	session := &model.Session{
		UserID:          userID,
		Username:        username,
		Name:            name,
		Code:            code,
		Email:           email,
		BiometricLogin:  biometricLogin,
		BiometricEdit:   biometricEdit,
		BiometricRemove: biometricRemove,
		Token:           tokenString,
		CreatedAt:       now,
		UpdatedAt:       now,
		Status:          "A",
		ExpiresAt:       now.Add(time.Hour * 24),
	}

	if planID.Valid {
		subscriptionPlan, err := sr.getSubscriptionPlan(planID.Int64)
		if err == nil {
			session.SubscriptionPlan = subscriptionPlan
		}
	} else {
		subscriptionPlan, err := sr.getSubscriptionPlan(1)
		if err == nil {
			session.SubscriptionPlan = subscriptionPlan
		}
	}

	err = sr.db.QueryRow(`
		insert into barrel.sessions(id
								   ,user_id
								   ,token
								   ,created_at
								   ,updated_at
								   ,status
								   ,expires_at)
							values (nextval('barrel.seq_sessions')
								   ,$1::bigint
								   ,$2::text
								   ,$3::timestamp
								   ,$4::timestamp
								   ,$5::bpchar
								   ,$6::timestamp)
							returning id
	`, session.UserID, session.Token, session.CreatedAt, session.UpdatedAt, session.Status, session.ExpiresAt).Scan(&session.ID)

	if err != nil {
		return nil, err
	}

	return session, nil
}
