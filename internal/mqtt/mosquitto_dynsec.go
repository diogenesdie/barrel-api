// mqtt/mosquitto_dynsec.go
package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MosquittoDynSecProvisioner struct {
	BrokerURL string // ex: tcp://localhost:1883
	AdminUser string
	AdminPass string
	ClientID  string
	Topic     string
}

func NewMosqDynSec(brokerURL, adminUser, adminPass string) *MosquittoDynSecProvisioner {
	return &MosquittoDynSecProvisioner{
		BrokerURL: brokerURL,
		AdminUser: adminUser,
		AdminPass: adminPass,
		Topic:     "$CONTROL/dynamic-security/v1",
		ClientID:  "barrel-admin",
	}
}

func (p *MosquittoDynSecProvisioner) pub(payload any) error {
	opts := mqtt.NewClientOptions().AddBroker(p.BrokerURL).
		SetClientID(p.ClientID).
		SetUsername(p.AdminUser).
		SetPassword(p.AdminPass).
		SetAutoReconnect(true)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	defer c.Disconnect(100)

	b, _ := json.Marshal(map[string]any{"commands": []any{payload}})
	tok := c.Publish(p.Topic, 0, false, b)
	if !tok.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout to dynsec topic")
	}
	return tok.Error()
}

func (p *MosquittoDynSecProvisioner) CreateUser(ctx context.Context, u User) error {
	cmd := map[string]any{
		"command":  "createClient",
		"username": u.Username,
		"password": u.Password,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) UpdatePassword(ctx context.Context, username, newPassword string) error {
	cmd := map[string]any{
		"command":  "setClientPassword",
		"username": username,
		"password": newPassword,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) DeleteUser(ctx context.Context, username string) error {
	cmd := map[string]any{
		"command":  "deleteClient",
		"username": username,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) CreateRole(ctx context.Context, role string) error {
	cmd := map[string]any{
		"command":  "createRole",
		"rolename": role,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) AddRoleACL(ctx context.Context, role string, aclType string, topic string) error {
	cmd := map[string]any{
		"command":  "addRoleACL",
		"rolename": role,
		"acltype":  aclType,
		"topic":    topic,
		"allow":    true,
		"priority": 0,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) AddClientRole(ctx context.Context, username, role string) error {
	cmd := map[string]any{
		"command":  "addClientRole",
		"username": username,
		"rolename": role,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) DeleteRole(ctx context.Context, role string) error {
	cmd := map[string]any{
		"command":  "deleteRole",
		"rolename": role,
	}
	return p.pub(cmd)
}

func (p *MosquittoDynSecProvisioner) RemoveRoleACL(ctx context.Context, role string, aclType string, topic string) error {
	cmd := map[string]any{
		"command":  "removeRoleACL",
		"rolename": role,
		"acltype":  aclType,
		"topic":    topic,
	}
	return p.pub(cmd)
}
