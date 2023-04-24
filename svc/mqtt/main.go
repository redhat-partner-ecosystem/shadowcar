package main

import (
	"crypto/tls"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/txsvc/stdlib/v2"
)

const (
	// expected ENV variables
	CLIENT_ID            = "client_id"
	SOURCE_TOPIC         = "source_topic"
	MQTT_HOST            = "mqtt_host"
	MQTT_PROTOCOL        = "mqtt_protocol"
	MQTT_PORT            = "mqtt_port"
	MQTT_TLS_SKIP_VERIFY = "mqtt_tls_skip_verify"
	MQTT_USER            = "default-mqtt-user"
	MQTT_PASSWORD        = "default-mqtt-password"

	// controll the behaviour of the listener
	traceMQTT = false // debug MQTT client
	debug     = true  // log all messages & events

	// MQTT QoS, see https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/
	AtMostOnce  byte = 0
	AtLeastOnce byte = 1
	ExactlyOnce byte = 2
)

var (
	// Cloud endpoints (Cloud-to-Device)
	mqttIntegrationHost     string
	mqttIntegrationProtocol string
	mqttIntegrationPort     string
	clientID                string
	sourceTopic             string

	// internal stuff
	shutdown bool = false
)

func init() {
	//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if traceMQTT {
		mqtt.CRITICAL = stdlog.New(os.Stdout, "[CRIT] ", 0)
		mqtt.WARN = stdlog.New(os.Stdout, "[WARN]  ", 0)
		mqtt.DEBUG = stdlog.New(os.Stdout, "[DEBUG] ", 0)
	}
	mqtt.ERROR = stdlog.New(os.Stdout, "[ERROR] ", 0)

	mqttIntegrationHost = stdlib.GetString(MQTT_HOST, "")
	mqttIntegrationProtocol = stdlib.GetString(MQTT_PROTOCOL, "tcp")
	mqttIntegrationPort = stdlib.GetString(MQTT_PORT, "1883")
	if mqttIntegrationHost == "" {
		panic(fmt.Errorf("missing env MQTT_HOST"))
	}
	clientID = stdlib.GetString(CLIENT_ID, "mqtt-listener-svc")
	sourceTopic = stdlib.GetString("source_topic", "")

	// setup shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		shutdown = true
		log.Warn().Msg("shutting down")
	}()

}

func main() {

	// listen for messages on the integration endpoint
	cl := createMqttClient(mqttIntegrationProtocol, mqttIntegrationHost, mqttIntegrationPort, clientID, stdlib.GetString(MQTT_USER, ""), stdlib.GetString(MQTT_PASSWORD, ""))
	if token := cl.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal().Err(token.Error()).Msg(token.Error().Error())
	}
	defer cl.Disconnect(250)
	cl.Subscribe(sourceTopic, AtLeastOnce, receiveMqttMsg)

	// background stuff goes here ...
	for !shutdown {
		time.Sleep(10 * time.Second)
	}
}

func receiveMqttMsg(client mqtt.Client, msg mqtt.Message) {
	log.Logger.Info().Str("topic", msg.Topic()).Str("body", string(msg.Payload())).Msg(fmt.Sprintf("message id %d", msg.MessageID()))
}

func createMqttClient(protocol, host, port, clientID, username, password string) mqtt.Client {
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
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Logger.Info().Bool("connected", c.IsConnected()).Bool("open", c.IsConnectionOpen()).Msg("onConnect")
	})

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
