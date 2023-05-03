package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redhat-partner-ecosystem/shadowcar/internal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/txsvc/stdlib/v2"
)

const (
	// expected ENV variables
	CLIENT_ID = "client_id"
	GROUP_ID  = "group_id"

	SOURCE_TOPIC = "source_topic"

	KAFKA_SERVICE      = "kafka_service"
	KAFKA_SERVICE_PORT = "kafka_service_port"
	KAFKA_AUTO_OFFSET  = "auto_offset"

	LOG_LEVEL_DEBUG = "log_level_debug"
)

type (
	CampaignMapping struct {
		CampaignID string `json:"campaignId,omitempty"`
		Zone       string `json:"zone,omitempty"`
	}

	Campaigns []CampaignMapping
)

var (
	kc *kafka.Consumer
)

func init() {

	// setup logging
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if internal.GetBool(LOG_LEVEL_DEBUG, false) {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// setup Kafka client
	clientID := stdlib.GetString(CLIENT_ID, "kafka-listener-svc")
	groupID := stdlib.GetString(GROUP_ID, "kafka-listener")
	autoOffset := stdlib.GetString(KAFKA_AUTO_OFFSET, "end") // smallest, earliest, beginning, largest, latest, end

	// kafka setup
	kafkaService := stdlib.GetString(KAFKA_SERVICE, "")
	if kafkaService == "" {
		panic(fmt.Errorf("missing env KAFKA_SERVICE"))
	}
	kafkaServicePort := stdlib.GetString(KAFKA_SERVICE_PORT, "9092")
	kafkaServer := fmt.Sprintf("%s:%s", kafkaService, kafkaServicePort)

	// https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md
	_kc, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":       kafkaServer,
		"client.id":               clientID,
		"group.id":                groupID,
		"connections.max.idle.ms": 0,
		"auto.offset.reset":       autoOffset,
		"broker.address.family":   "v4",
	})
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}
	kc = _kc

	// setup campaigns etc ...
	campaignsJSON := stdlib.GetString("campaigns", "")
	if campaignsJSON != "" {
		var campaigns Campaigns
		err = json.Unmarshal([]byte(campaignsJSON), &campaigns)
		if err != nil {
			log.Err(err).Msg("")
		}
		log.Info().Any("c", campaigns).Msg("campaigns")
	}

	// setup prometheus endpoint
	internal.StartPrometheusListener()
}

func main() {

	clientID := stdlib.GetString("client_id", "kafka-listener-svc")
	sourceTopic := stdlib.GetString("source_topic", "")

	// metrics collectors
	opsTxProcessed := promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_listener_events",
		Help: "The number of processed events",
	})

	// subscribe to the topic(s)
	err := kc.SubscribeTopics(strings.Split(sourceTopic, ","), nil)
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}

	log.Info().Str("source", sourceTopic).Str("clientid", clientID).Msg("start listening")

	for {
		msg, err := kc.ReadMessage(-1)

		if err == nil {
			var evt internal.ZoneChangeEvent
			err = json.Unmarshal(msg.Value, &evt)
			if err != nil {
				log.Err(err).Msg("")
			}

			// handle the event
			zoneChange(&evt)

			// update metrics
			opsTxProcessed.Inc()

		} else {
			// The client will automatically try to recover from all errors.
			log.Error().Err(err).Msg("error")
		}
	}
}

func zoneChange(evt *internal.ZoneChangeEvent) {
	log.Info().Str("vin", evt.CarID).Msg(evt.String())

	if evt.NextZoneID != "" {
		// only do sth in case a car enters a zone

	}
}