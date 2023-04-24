package internal

import (
	"crypto/tls"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

const (
	// https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/
	AtMostOnce  byte = 0
	AtLeastOnce byte = 1
	ExactlyOnce byte = 2
)

var (
	MqttEndpointHost        string = "mqtt-endpoint-ws-bobbycar.apps.sdv.sdv.luxoft.com"
	MqttEndpointProtocol    string = "wss"
	MqttEndpointPort        string = "443"
	MqttIntegrationHost     string = "mqtt-integration-ws-bobbycar.apps.sdv.sdv.luxoft.com"
	MqttIntegrationProtocol string = "wss"
	MqttIntegrationPort     string = "443"

	KafkaBootstrapService     string = "bobbycar-cluster-kafka-bootstrap-bobbycar.apps.sdv.sdv.luxoft.com"
	KafkaBootstrapServicePort string = "9092"
)

func CreateMqttClient(protocol, host, port, clientID, username, password string) mqtt.Client {
	// setup and configuration
	broker := fmt.Sprintf("%s://%s:%s", protocol, host, port)
	opts := mqtt.NewClientOptions().AddBroker(broker)

	opts.SetCleanSession(true)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(5 * time.Second)

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Logger.Info().Str("topic", msg.Topic()).Str("body", string(msg.Payload())).Msg(fmt.Sprintf("un-handled message id %d", msg.MessageID()))
	})
	opts.SetOnConnectHandler(onConnectHandler)

	if username != "" {

		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	// create a client
	return mqtt.NewClient(opts)
}

func onConnectHandler(c mqtt.Client) {
	log.Logger.Info().Bool("connected", c.IsConnected()).Bool("open", c.IsConnectionOpen()).Msg("onConnect")
}
