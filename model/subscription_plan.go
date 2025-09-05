package model

import "time"

type SubscriptionPlan struct {
	ID                 uint64    `json:"id"`
	Name               string    `json:"name"`
	Price              *float64  `json:"price"`
	Currency           string    `json:"currency"`
	MaxDevices         *int      `json:"max_devices"`
	LocalCommunication bool      `json:"local_communication"`
	MQTTIncluded       bool      `json:"mqtt_included"`
	TechnicalSupport   bool      `json:"technical_support"`
	PrioritySupport    bool      `json:"priority_support"`
	CustomWidgets      bool      `json:"custom_widgets"`
	CustomIntegrations bool      `json:"custom_integrations"`
	IsPopular          bool      `json:"is_popular"`
	IsCustomPrice      bool      `json:"is_custom_price"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
