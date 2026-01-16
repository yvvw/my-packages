package internal

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	mq mqtt.Client
}

func New(option *mqtt.ClientOptions) *Client {
	return &Client{
		mq: mqtt.NewClient(option),
	}
}

func NewSimple(server string, token string) *Client {
	option := mqtt.NewClientOptions().
		AddBroker(server).
		SetClientID(token).
		SetOnConnectHandler(func(_ mqtt.Client) {
			log.WithFields(log.Fields{"server": server, "token": token}).Info("connected")
		})
	return New(option)
}

func (c *Client) Connect() error {
	if t := c.mq.Connect(); t.WaitTimeout(time.Second*5) && t.Error() != nil {
		return t.Error()
	}
	return nil
}

func (c *Client) Disconnect() {
	c.mq.Disconnect(5_000)
}

func (c *Client) Subscribe(topic string, callback mqtt.MessageHandler) error {
	if token := c.mq.Subscribe(topic, byte(0), callback); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) Unsubscribe(topics ...string) error {
	if token := c.mq.Unsubscribe(topics...); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
