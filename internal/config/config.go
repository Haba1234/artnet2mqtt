package config

import (
	"github.com/BurntSushi/toml"
)

// Config структура конфигурации.
type Config struct {
	Logger LogConf  // Logger - конфигурация регистратора.
	MQTT   MQTTConf // MQTT - конфигурация MQTT клиента.
}

// LogConf структура конфигурации.
type LogConf struct {
	Level string `toml:"log-level"` // Level - уровень логирования.
}

// MQTTConf структура конфигурации.
type MQTTConf struct {
	ClientID string `toml:"clientID"` // ClientID - имя клиента.
	// TODO Schema   string `toml:"schema"`   // Schema - тип подключения.
	Host     string `toml:"server"`   // Host - адрес MQTT сервера.
	Port     string `toml:"port"`     // Port - порт MQTT сервера.
	User     string `toml:"user"`     // User - логин для подключения к MQTT серверу.
	Password string `toml:"password"` // Password - пароль для подключения к MQTT серверу.
	Qos      byte   `toml:"qos"`      // Qos - качество обслуживания.
}

// NewConfig конструктор.
func NewConfig(path string) (*Config, error) {
	// default values
	cfg := Config{
		Logger: LogConf{},
		MQTT:   MQTTConf{},
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return &cfg, err
	}
	return &cfg, nil
}
