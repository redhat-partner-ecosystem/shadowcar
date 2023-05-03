package internal

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"

	"github.com/txsvc/stdlib/v2"
)

const (
	// https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/
	AtMostOnce  byte = 0
	AtLeastOnce byte = 1
	ExactlyOnce byte = 2

	PROMETHEUS_HOST         = "prometheus_host"
	PROMETHEUS_METRICS_PATH = "prometheus_metrics_path"
)

func StartPrometheusListener() {
	// prometheus endpoint setup
	promHost := stdlib.GetString(PROMETHEUS_HOST, "0.0.0.0:2112")
	promMetricsPath := stdlib.GetString(PROMETHEUS_METRICS_PATH, "/metrics")

	// start the metrics listener
	go func() {
		log.Debug().Msg("start metrics")

		http.Handle(promMetricsPath, promhttp.Handler())
		http.ListenAndServe(promHost, nil)
	}()
}

func Duration(d time.Duration, dicimal int) time.Duration {
	shift := int(math.Pow10(dicimal))

	units := []time.Duration{time.Second, time.Millisecond, time.Microsecond, time.Nanosecond}
	for _, u := range units {
		if d > u {
			div := u / time.Duration(shift)
			if div == 0 {
				break
			}
			d = d / div * div
			break
		}
	}
	return d
}

func XID() string {
	return xid.New().String()
}

// FIXME move this to stdlib
func GetBool(env string, def bool) bool {
	e, ok := os.LookupEnv(env)
	if !ok {
		return def
	}

	e = strings.ToLower(e)
	if e == "true" || e == "yes" || e == "1" {
		return true
	}
	return false
}

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
