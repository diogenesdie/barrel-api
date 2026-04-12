package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	User     string
	Password string
	Name     string
}

type Config struct {
	Database      DatabaseConfig
	ServerAddress string
	Token         string
	JWTSecret     string
	MQTTPassword       string
	MQTTBrokerURL      string
	MQTTAdminUser      string
	MQTTPublisherUser  string
	MQTTPublisherPass  string
}

func getEnv(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)

	if !ok {
		return defaultValue
	}

	return value
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dbConfig := DatabaseConfig{
		User:     getEnv("DB_USER", ""),
		Password: getEnv("DB_PASSWORD", ""),
		Name:     getEnv("DB_NAME", ""),
	}

	return &Config{
		Database:      dbConfig,
		ServerAddress: getEnv("SERVER_ADDRESS", ":3000"),
		Token:         getEnv("TOKEN", ""),
		JWTSecret:     getEnv("JWT_SECRET", "diogepepe_30_04_2024"),
		MQTTPassword:      getEnv("MQTT_ADMIN_PASS", "adminpass"),
		MQTTBrokerURL:     getEnv("MQTT_BROKER_URL", "tcp://dioge.com.br:1883"),
		MQTTAdminUser:     getEnv("MQTT_ADMIN_USER", "admin"),
		MQTTPublisherUser: getEnv("MQTT_PUBLISHER_USER", "barrel-api-publisher"),
		MQTTPublisherPass: getEnv("MQTT_PUBLISHER_PASS", ""),
	}
}
