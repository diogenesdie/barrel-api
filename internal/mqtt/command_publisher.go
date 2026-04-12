package mqtt

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// CommandPublisher publica comandos cifrados para dispositivos via MQTT.
// Replica a mesma lógica AES-CBC do app Flutter (crypto_utils.dart).
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

// PublishDeviceCommand cifra cmd com AES-CBC usando ivKey no formato "keyHex:ivHex"
// e publica no tópico users/{ownerUsername}/{deviceID}/command.
// deviceID é o identificador de firmware (SmartDevice.DeviceID), não o ID numérico do banco.
func (p *CommandPublisher) PublishDeviceCommand(ownerUsername, deviceID, ivKey, cmd string) error {
	encrypted, err := encryptAESCBC(ivKey, cmd)
	if err != nil {
		return fmt.Errorf("encrypt command: %w", err)
	}

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

	tok := c.Publish(topic, 1, false, encrypted)
	if !tok.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("mqtt publish timeout to %s", topic)
	}
	return tok.Error()
}

// encryptAESCBC implementa AES-CBC + PKCS7 + Base64, idêntico ao Flutter.
// ivKey formato: "keyHex:ivHex"
func encryptAESCBC(ivKey, plaintext string) (string, error) {
	parts := strings.SplitN(ivKey, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("ivKey inválido: esperado 'keyHex:ivHex', recebido '%s'", ivKey)
	}

	keyBytes, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("decodificar chave hex: %w", err)
	}

	ivBytes, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decodificar iv hex: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("criar cipher AES: %w", err)
	}

	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, ivBytes).CryptBlocks(ciphertext, padded)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - (len(data) % blockSize)
	padded := make([]byte, len(data)+pad)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(pad)
	}
	return padded
}
