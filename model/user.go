package model

import "time"

type User struct {
	ID              uint64    `json:"id"`
	Type            string    `json:"type"`
	Username        string    `json:"username"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Code            string    `json:"code"`
	BiometricLogin  bool      `json:"biometric_login"`
	BiometricEdit   bool      `json:"biometric_edit"`
	BiometricRemove bool      `json:"biometric_remove"`
	Password        *string   `json:"password"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpdateUserProfileRequest struct {
	Name            *string `json:"name"`
	Email           *string `json:"email"`
	BiometricLogin  *bool   `json:"biometric_login"`
	BiometricEdit   *bool   `json:"biometric_edit"`
	BiometricRemove *bool   `json:"biometric_remove"`
}
