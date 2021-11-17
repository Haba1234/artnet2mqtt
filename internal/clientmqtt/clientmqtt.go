package clientmqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"artnet2mqtt/internal/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ClientMQTT структура клиента MQTT.
type ClientMQTT struct {
	ctx       context.Context
	log       logger.Logger
	cfgClient MQTTConf
	client    mqtt.Client
	opts      *mqtt.ClientOptions
	dmxDataCh chan<- DataCh
	topics    map[nameTopic]dmxAddr
}

// MQTTClient is a convenience interface to use within this application.
type MQTTClient interface {
	Start(ctx context.Context, dmxDataCh chan<- DataCh) error
	Stop() error
	PubTopic() error
}

// NewClient конструктор.
func NewClient(log logger.Logger, cfgClient MQTTConf) *ClientMQTT {
	return &ClientMQTT{
		log:       log,
		cfgClient: cfgClient,
		topics:    map[nameTopic]dmxAddr{},
	}
}

func (c *ClientMQTT) Start(ctx context.Context, dmxDataCh chan DataCh) error {
	// TODO перенаправить в logger
	if c.log.GetLevel() == "debug" {
		mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
		mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
		mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	}

	c.ctx = ctx
	c.dmxDataCh = dmxDataCh

	// TODO добавить проверки по брокеру на ошибки конфигурации.
	c.opts = mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("%s://%s:%s", c.cfgClient.Schema, c.cfgClient.Host, c.cfgClient.Port)).
		SetUsername(c.cfgClient.User).
		SetPassword(c.cfgClient.Password).
		SetDefaultPublishHandler(c.messageHandler).
		SetOnConnectHandler(c.connectHandler).
		SetConnectionLostHandler(c.connectLostHandler).
		SetClientID(c.cfgClient.ClientID).
		SetOrderMatters(false).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second). // TODO добавить в конфиг.
		SetMaxReconnectInterval(5 * time.Second).
		SetKeepAlive(30 * time.Second)

	c.client = mqtt.NewClient(c.opts)

	token := c.client.Connect()
	select {
	case <-token.Done():
		if token.Error() != nil {
			return token.Error()
		}
		break
	case <-c.ctx.Done():
		return errors.New("context canceled")
	}

	c.log.With(logger.Fields{"module": "mqtt"}).Infof("Status: %v", c.client.IsConnected())
	return nil
}

func (c *ClientMQTT) Stop() error {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(500)
	}
	return nil
}

func (c *ClientMQTT) connectHandler(_ mqtt.Client) {
	c.log.With(logger.Fields{"module": "mqtt"}).Info("client connected to server")
}

func (c *ClientMQTT) connectLostHandler(_ mqtt.Client, err error) {
	c.log.With(logger.Fields{"module": "mqtt"}).Errorf("server connect lost: %v\n", err)
}

func (c *ClientMQTT) messageHandler(_ mqtt.Client, msg mqtt.Message) {
	c.log.With(logger.Fields{"module": "mqtt"}).Debugf("received message: %v from topic: %s", msg.Payload(), msg.Topic())
	go c.sendDataToArtNet(msg)
}

func (c *ClientMQTT) sendDataToArtNet(msg mqtt.Message) {
	message := msg
	addr, ok := c.topics[nameTopic(message.Topic())]
	if !ok {
		c.log.With(logger.Fields{"module": "mqtt"}).Error("accepted topic was not found in the database. Recording in Art-net was canceled")
		return
	}

	var data Payload
	if err := json.Unmarshal(message.Payload(), &data); err != nil {
		c.log.With(logger.Fields{"module": "mqtt"}).Errorf("message could not be parsed (%v): %v\n", message.Payload(), err)
		return
	}
	c.log.With(logger.Fields{"module": "mqtt"}).Debugf("message payload parsed. Result: %v\n", data)
	c.dmxDataCh <- DataCh{Addr: uint16(addr), Data: data}
}

func (c *ClientMQTT) sub(topic string) {
	token := c.client.Subscribe(topic, 0, nil)
	go func() {
		topic := topic
		token := token
		select {
		case <-c.ctx.Done():
			return
		case <-token.Done():
			if token.Error() != nil {
				c.log.With(logger.Fields{"module": "mqtt"}).Errorf("topic %s subscription error. %v\n", topic, token.Error())
				return
			}
		}
		c.log.With(logger.Fields{"module": "mqtt"}).Debugf("topic %s subscribed\n", topic)
	}()
}

func (c *ClientMQTT) PubTopic(topic string, out uint16) {
	if _, ok := c.topics[nameTopic(topic)]; ok {
		c.log.With(logger.Fields{"module": "mqtt"}).Debug("topic существует:", topic)
		return
	}
	c.topics[nameTopic(topic)] = dmxAddr(out)
	msg, err := json.Marshal(Payload{
		DMXCommand{Channel: 0, Value: 0}, DMXCommand{Channel: 1, Value: 0}, DMXCommand{Channel: 2, Value: 0},
	})
	if err != nil {
		c.log.With(logger.Fields{"module": "mqtt"}).Errorf("public topic. msg: %v", err)
	}
	// TODO добавить qos и другие настройки.
	token := c.client.Publish(topic, 0, false, msg)
	go func() {
		topic := topic
		token := token
		select {
		case <-c.ctx.Done():
			return
		case <-token.Done():
			if token.Error() != nil {
				c.log.With(logger.Fields{"module": "mqtt"}).Errorf("error publish topic %s. %v\n", topic, token.Error())
				return
			}
			c.sub(topic)
		}
	}()
}
