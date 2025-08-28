// mqtt/provisioner.go
package mqtt

import "context"

type User struct {
	Username string
	Password string
}

type Provisioner interface {
	CreateUser(ctx context.Context, u User) error
	UpdatePassword(ctx context.Context, username, newPassword string) error
	DeleteUser(ctx context.Context, username string) error
	CreateRole(ctx context.Context, role string) error
	AddRoleACL(ctx context.Context, role string, aclType string, topic string) error
	AddClientRole(ctx context.Context, username, role string) error
	DeleteRole(ctx context.Context, role string) error
}
