package model

import "time"

type OAuthCode struct {
	ID          uint64    `json:"id"`
	Code        string    `json:"code"`
	UserID      uint64    `json:"user_id"`
	RedirectURI string    `json:"redirect_uri"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
	CreatedAt   time.Time `json:"created_at"`
}
