package mqtt

import (
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// CommandPublisher publica comandos para dispositivos via MQTT.
// O firmware espera texto puro no tópico MQTT (sem criptografia),
// diferente do HTTP local que usa AES-CBC.
type CommandPublisher struct {
	BrokerURL string
	AdminUser string
	AdminPass string
}

func NewCommandPublisher(brokerURL, adminUser, adminPass string) *CommandPublisher {
	return &CommandPublisher{
		BrokerURL: brokerURL,
		AdminUser: adminUser,
		AdminPass: adminPass,
	}
}

// PublishDeviceCommand publica cmd em texto puro no tópico
// users/{ownerUsername}/{deviceID}/command.
// deviceID é o identificador de firmware (SmartDevice.DeviceID), não o ID numérico do banco.
func (p *CommandPublisher) PublishDeviceCommand(ownerUsername, deviceID, ivKey, cmd string) error {
	topic := fmt.Sprintf("users/%s/%s/command", ownerUsername, deviceID)

	opts := pahomqtt.NewClientOptions().
		AddBroker(p.BrokerURL).
		SetClientID("barrel-cmd-publisher").
		SetUsername(p.AdminUser).
		SetPassword(p.AdminPass).
		SetAutoReconnect(false).
		SetConnectTimeout(5 * time.Second)

	c := pahomqtt.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return fmt.Errorf("mqtt connect: %w", tok.Error())
	}
	defer c.Disconnect(200)

	tok := c.Publish(topic, 1, false, cmd)
	if !tok.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("mqtt publish timeout to %s", topic)
	}
	return tok.Error()
}
