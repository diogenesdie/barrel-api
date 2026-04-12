package repository

import (
	"barrel-api/model"
	"database/sql"
	"errors"
	"time"
)

var ErrOAuthCodeNotFound = errors.New("oauth code not found")
var ErrOAuthCodeExpired = errors.New("oauth code expired")
var ErrOAuthCodeUsed = errors.New("oauth code already used")

type OAuthRepository struct {
	db *sql.DB
}

func NewOAuthRepository(db *sql.DB) *OAuthRepository {
	return &OAuthRepository{db}
}

func (r *OAuthRepository) CreateCode(code *model.OAuthCode) error {
	return r.db.QueryRow(`
		INSERT INTO barrel.oauth_codes (code, user_id, redirect_uri, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, code.Code, code.UserID, code.RedirectURI, code.ExpiresAt).
		Scan(&code.ID, &code.CreatedAt)
}

// ConsumeCode atomically marks the code as used and returns the OAuthCode.
// Returns ErrOAuthCodeNotFound, ErrOAuthCodeExpired or ErrOAuthCodeUsed on failure.
func (r *OAuthRepository) ConsumeCode(code, redirectURI string) (*model.OAuthCode, error) {
	oc := &model.OAuthCode{}
	err := r.db.QueryRow(`
		UPDATE barrel.oauth_codes
		   SET used = TRUE
		 WHERE code         = $1
		   AND redirect_uri = $2
		   AND used         = FALSE
		   AND expires_at   > now()
		RETURNING id, code, user_id, redirect_uri, expires_at, used, created_at
	`, code, redirectURI).Scan(
		&oc.ID, &oc.Code, &oc.UserID, &oc.RedirectURI,
		&oc.ExpiresAt, &oc.Used, &oc.CreatedAt,
	)

	if err == sql.ErrNoRows {
		// Determine why it failed
		var used bool
		var expiresAt time.Time
		row := r.db.QueryRow(`
			SELECT used, expires_at FROM barrel.oauth_codes
			 WHERE code = $1 AND redirect_uri = $2
		`, code, redirectURI)
		scanErr := row.Scan(&used, &expiresAt)
		if scanErr == sql.ErrNoRows {
			return nil, ErrOAuthCodeNotFound
		}
		if used {
			return nil, ErrOAuthCodeUsed
		}
		if expiresAt.Before(time.Now()) {
			return nil, ErrOAuthCodeExpired
		}
		return nil, ErrOAuthCodeNotFound
	}

	if err != nil {
		return nil, err
	}

	return oc, nil
}
