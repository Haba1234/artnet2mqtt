package clientmqtt

type MQTTConf struct {
	ClientID string // ClientID - уникальное имя клиента для брокеров.
	Schema   string // Schema - тип подключения.
	Host     string // Host - адрес MQTT сервера.
	Port     string // Port - порт MQTT сервера.
	User     string // User - логин для подключения к MQTT серверу.
	Password string // Password - пароль для подключения к MQTT серверу.
}

type nameTopic string
type dmxAddr uint16

type DataCh struct {
	Addr uint16
	Data Payload
}

type DMXCommand struct {
	Channel uint16 // Channel is the channel a command can talk to (0-511).
	Value   uint8  // Value is the value a DMX channel can represent (0-255).
}

type Payload []DMXCommand
