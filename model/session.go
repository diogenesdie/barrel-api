package model

import (
	"time"
)

type Session struct {
	ID               uint64            `json:"id"`
	UserID           uint64            `json:"user_id"`
	Username         string            `json:"username"`
	Name             string            `json:"name"`
	Code             string            `json:"code"`
	Email            string            `json:"email"`
	Token            string            `json:"token"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	Status           string            `json:"status"`
	SubscriptionPlan *SubscriptionPlan `json:"subscription_plan,omitempty"`
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
