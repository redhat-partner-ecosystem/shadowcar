package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/redhat-partner-ecosystem/shadowcar/internal"
	"github.com/rs/zerolog/log"

	"github.com/txsvc/stdlib/v2"
)

const (
	application = "bobbycar" // the Drogue application
	VIN         = "test-car1"
	//VIN = "WBAFR9C59BC270614"

	// endpoints
	carPostionQueue string = "car"        // channel for position data
	carMetricsQueue string = "carMetrics" // channel for other telemetry e.g. speed

	// integration topics
	deviceToCloudTopic string = "app/bobbycar"
	cloudToDeviceTopic string = "command/%s/%s/%s"

	// authentication
	defaultDevicePassword = "car123456"
	defaultAdminUser      = "admin"
	defaultAdminToken     = "drg_0r0qEl_gkhoMbI4FKRNDUGh2o1Xtart4FPcS63pEX5Q"

	// MQTT QoS, see https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/
	AtMostOnce  byte = 0
	AtLeastOnce byte = 1
	ExactlyOnce byte = 2

	LOG_LEVEL_DEBUG      = "log_level_debug"
	LOG_LEVEL_MQTT_TRACE = "log_level_mqtt_trace"
)

var (
	// Device endpoints (Device-to-Cloud)
	MqttEndpointHost     string = "mqtt-endpoint-ws-bobbycar.apps.sdv.sdv.luxoft.com"
	MqttEndpointProtocol string = "wss"
	MqttEndpointPort     string = "443"

	// Cloud endpoints (Cloud-to-Device)
	MqttIntegrationHost     string = "mqtt-integration-ws-bobbycar.apps.sdv.sdv.luxoft.com"
	MqttIntegrationProtocol string = "wss"
	MqttIntegrationPort     string = "443"

	// internal stuff
	shutdown bool = false
	gpx      [][]float32
)

type (
	/* Example:
	{
		"carid":"",
		"EventTime": 1682057455,
		"elev": "0.0",
		"lat": 39.78876,
		"long": -86.23759
	}
	*/
	Coordinates struct {
		VIN       string  `json:"carid"`
		EventTime int64   `json:"eventTime"`
		Elev      string  `json:"elev"`
		Lat       float32 `json:"lat"`
		Long      float32 `json:"long"`
	}
)

func (co *Coordinates) String() string {
	return fmt.Sprintf("%s,[%f,%f]", co.VIN, co.Lat, co.Long)
}

func init() {
	stdlib.Seed(stdlib.Now())

	// setup logging
	internal.SetLogLevel()

	if internal.GetBool(LOG_LEVEL_MQTT_TRACE, false) {
		mqtt.CRITICAL = stdlog.New(os.Stdout, "[CRIT] ", 0)
		mqtt.WARN = stdlog.New(os.Stdout, "[WARN]  ", 0)
		mqtt.DEBUG = stdlog.New(os.Stdout, "[DEBUG] ", 0)
	}
	mqtt.ERROR = stdlog.New(os.Stdout, "[ERROR] ", 0)

	// setup shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		shutdown = true
		log.Warn().Msg("shutting down")
	}()

	gpx = make([][]float32, 10)
	gpx[0] = make([]float32, 2)
	gpx[0][0] = 39.79322767551741
	gpx[0][1] = -86.23885000569480
	gpx[1] = make([]float32, 2)
	gpx[1][0] = 39.78876058936334
	gpx[1][1] = -86.23758874171043
	gpx[2] = make([]float32, 2)
	gpx[2][0] = 39.78835427615011
	gpx[2][1] = -86.23288199536968
	gpx[3] = make([]float32, 2)
	gpx[3][0] = 39.79057400070364
	gpx[3][1] = -86.23037783076391
	gpx[4] = make([]float32, 2)
	gpx[4][0] = 39.79407519606684
	gpx[4][1] = -86.23046930502242
	gpx[5] = make([]float32, 2)
	gpx[5][0] = 39.79820422186210
	gpx[5][1] = -86.23059328183015
	gpx[6] = make([]float32, 2)
	gpx[6][0] = 39.80188253584900
	gpx[6][1] = -86.23310398871426
	gpx[7] = make([]float32, 2)
	gpx[7][0] = 39.80176978816294
	gpx[7][1] = -86.23694258379227
	gpx[8] = make([]float32, 2)
	gpx[8][0] = 39.79807508553657
	gpx[8][1] = -86.23897694127818
	gpx[9] = make([]float32, 2)
	gpx[9][0] = 39.79372139852681
	gpx[9][1] = -86.23886288923050
}

func main() {

	// simulate cars ...
	vin := stdlib.GetString("VIN", VIN)
	go simulate(vin, application)

	/*
		if debug {
			// listen for messages on the integration endpoint
			cl := createMqttClient(MqttIntegrationProtocol, MqttIntegrationHost, MqttIntegrationPort, application, defaultAdminUser, defaultAdminToken)
			if token := cl.Connect(); token.Wait() && token.Error() != nil {
				log.Fatal().Err(token.Error()).Msg(token.Error().Error())
			}
			defer cl.Disconnect(250)
			cl.Subscribe(deviceToCloudTopic, AtLeastOnce, receiveMqttMsg)
		}
	*/

	// background stuff goes here ...
	for !shutdown {
		time.Sleep(10 * time.Second)
	}
}

func simulate(vin, application string) {
	user := fmt.Sprintf("%s-gw@%s", vin, application)
	log.Info().Msg(fmt.Sprintf("simulating car with VIN='%s'", vin))

	// connect to the endpoint gateway

	cl := createMqttClient(MqttEndpointProtocol, MqttEndpointHost, MqttEndpointPort, vin, user, defaultDevicePassword)
	if token := cl.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal().Err(token.Error()).Msg(token.Error().Error())
	}
	defer cl.Disconnect(250)

	// listen for commands
	// see https://book.drogue.io/drogue-cloud/dev/user-guide/endpoint-mqtt.html#_subscribe_to_commands
	topic := "command/inbox//#"
	if token := cl.Subscribe(topic, AtLeastOnce, receiveCommand); token.Wait() && token.Error() != nil {
		log.Fatal().Err(token.Error()).Msg(token.Error().Error())
	}

	// main event loop

	tick := 0
	lastTimestamp := int64(0)
	for !shutdown {
		// simulate the car driving
		df := drive(vin, tick, lastTimestamp)

		// publish coordinates
		payload, _ := json.Marshal(df)
		if token := cl.Publish(fmt.Sprintf("%s/%s", carPostionQueue, df.VIN), AtMostOnce, false, payload); token.Wait() && token.Error() != nil {
			log.Fatal().Err(token.Error()).Msg(token.Error().Error())
		}

		//if log.Debug().Enabled() {
		log.Debug().Str("vin", vin).Str("pos", df.String()).Msg(fmt.Sprintf("message #%d", tick))
		//}

		// housekeeping
		tick++
		lastTimestamp = df.EventTime

		// drive slowly ...
		time.Sleep(4 * time.Second)
	}

	log.Warn().Msg(fmt.Sprintf("stopping car with VIN='%s'", vin))
}

func drive(vin string, tick int, timestamp int64) Coordinates {
	// simulate driving by sending predefined coordinates
	idx := tick % 10
	return Coordinates{
		VIN:       vin,
		EventTime: stdlib.Now(),
		Elev:      "0.0",
		Lat:       gpx[idx][0],
		Long:      gpx[idx][1],
	}
}

func receiveCommand(client mqtt.Client, msg mqtt.Message) {
	log.Logger.Info().Str("topic", msg.Topic()).Str("cmd", string(msg.Payload())).Msg(fmt.Sprintf("message id %d", msg.MessageID()))
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
