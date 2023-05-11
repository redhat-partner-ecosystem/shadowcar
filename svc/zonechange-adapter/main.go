package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/rs/zerolog/log"

	"github.com/txsvc/apikit"
	"github.com/txsvc/apikit/api"
	"github.com/txsvc/stdlib/v2"

	"github.com/redhat-partner-ecosystem/shadowcar/api/drogue"
	"github.com/redhat-partner-ecosystem/shadowcar/api/ota"
	"github.com/redhat-partner-ecosystem/shadowcar/internal"
)

const (
	// expected ENV variables
	CLIENT_ID      = "client_id"
	GROUP_ID       = "group_id"
	APPLICATION_ID = "application_id"

	SOURCE_TOPIC = "source_topic"

	KAFKA_SERVICE      = "kafka_service"
	KAFKA_SERVICE_PORT = "kafka_service_port"
	KAFKA_AUTO_OFFSET  = "auto_offset"

	DefaultTTL = time.Minute * 1
)

var (
	campaigns           []string          // list of known campaigns we care about
	nextCampaignMapping map[string]string // current campaign -> next campaign
	campaignZoneMapping map[string]string

	kc *kafka.Consumer
	cm *ota.CampaignManagerClient
	dm *drogue.DrogueClient
)

func init() {

	// setup logging
	internal.SetLogLevel()

	// setup Kafka client
	clientID := stdlib.GetString(CLIENT_ID, "kafka-listener-svc")
	groupID := stdlib.GetString(GROUP_ID, "kafka-listener")
	autoOffset := stdlib.GetString(KAFKA_AUTO_OFFSET, "end") // smallest, earliest, beginning, largest, latest, end

	// kafka setup
	kafkaService := stdlib.GetString(KAFKA_SERVICE, "")
	if kafkaService == "" {
		log.Fatal().Err(fmt.Errorf("missing env KAFKA_SERVICE")).Msg("aborting")
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

	// campaign manager client
	cm, err = ota.NewCampaignManagerClient(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}

	// drogue client
	dm, err = drogue.NewDrogueClient(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}

	// HACK
	campaigns = make([]string, 4)
	campaigns[0] = "aaaaaaaa-0000-0000-0000-000000000000" // Summit Adaptive Autosar Update A
	campaigns[1] = "bbbbbbbb-0000-0000-0000-000000000000" // Summit Adaptive Autosar Update B
	campaigns[2] = "00000000-0000-0000-0000-aaaaaaaaaaaa" // VECS Adaptive Autosar Update A
	campaigns[3] = "00000000-0000-0000-0000-bbbbbbbbbbbb" // VECS Adaptive Autosar Update B

	nextCampaignMapping = make(map[string]string)
	nextCampaignMapping["aaaaaaaa-0000-0000-0000-000000000000"] = "bbbbbbbb-0000-0000-0000-000000000000"
	nextCampaignMapping["bbbbbbbb-0000-0000-0000-000000000000"] = "aaaaaaaa-0000-0000-0000-000000000000"
	nextCampaignMapping["00000000-0000-0000-0000-aaaaaaaaaaaa"] = "00000000-0000-0000-0000-bbbbbbbbbbbb"
	nextCampaignMapping["00000000-0000-0000-0000-bbbbbbbbbbbb"] = "00000000-0000-0000-0000-aaaaaaaaaaaa"

	campaignZoneMapping = make(map[string]string)
	campaignZoneMapping["aaaaaaaa-0000-0000-0000-000000000000"] = "luxoft"
	campaignZoneMapping["bbbbbbbb-0000-0000-0000-000000000000"] = "redhat"
	campaignZoneMapping["00000000-0000-0000-0000-aaaaaaaaaaaa"] = "luxoft"
	campaignZoneMapping["00000000-0000-0000-0000-bbbbbbbbbbbb"] = "redhat"

	// END HACK

	// setup campaigns and other structs ...

	go func() {
		for {
			refreshVehicleCampaignStatus()
			time.Sleep(60 * time.Second) // refesh every 60 sec
		}
	}()

}

func main() {
	// start the kafka event listener
	go listenZoneChangeEvents()

	// start the http listener
	svc, err := apikit.New(setup, shutdown)
	if err != nil {
		log.Fatal().Err(err).Msg(err.Error())
	}
	svc.Listen("")
}

func listenZoneChangeEvents() {

	clientID := stdlib.GetString("client_id", "kafka-listener-svc")
	sourceTopic := stdlib.GetString("source_topic", "")

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
			handleZoneChange(&evt)

		} else {
			// The client will automatically try to recover from all errors.
			log.Error().Err(err).Msg("error")
		}
	}
}

func handleZoneChange(evt *internal.ZoneChangeEvent) {

	device := lookupVehicle(evt.CarID)

	if device == nil {
		log.Warn().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Msg("device not found")
		return
	}

	// only do sth in case a car ENTERS a zone
	if evt.NextZoneID != "" {

		var age int64 = 1000 // just > zone_change_delay

		if last, ok := device.GetAnnotation("lastCampaignExecution"); ok {
			lastCampaignExecution, _ := strconv.ParseInt(last, 0, 64)
			age = stdlib.Now() - lastCampaignExecution
		}

		if age > stdlib.GetInt("zone_change_delay", 60) {

			currentCampaign, _ := device.GetAnnotation("campaign")
			campaign := nextCampaignMapping[currentCampaign]
			zone := campaignZoneMapping[campaign]

			log.Info().Str("vin", evt.CarID).Str("zone", zone).Str("campaign", campaign).Int64("age", age).Msg("executing campaign")

			err := cm.ExecuteCampaign(campaign)

			if err != nil {
				log.Error().Str("vin", evt.CarID).Str("zone", zone).Str("campaign", campaign).Err(err).Msg("execute campaign failed")
			} else {
				device.Metadata.Generation++
				device.SetLabel("zone", zone)
				device.SetAnnotation("lastCampaignExecution", fmt.Sprintf("%d", stdlib.Now()))
				device.SetAnnotation("campaign", campaign)

				dm.UpdateDevice(stdlib.GetString(APPLICATION_ID, "bobbycar"), device, false)
			}

		} else {
			log.Info().Str("vin", evt.CarID).Int64("age", age).Msg("ignoring zone trigger")
		}
	}
}

func lookupVehicle(vin string) *drogue.Device {
	status, device := dm.GetDevice(stdlib.GetString(APPLICATION_ID, "bobbycar"), vin)
	if status != http.StatusOK {
		return nil
	}
	return &device
}

func refreshVehicleCampaignStatus() {
	for _, campaignId := range campaigns {
		status, exec := cm.GetCampaignExecution(campaignId)
		if status == http.StatusOK {
			if len(exec) > 0 {
				for _, e := range exec {
					device := lookupVehicle(e.VIN)
					if device != nil {
						device.SetAnnotation("campaign", e.CampaignID)
						device.SetAnnotation("campaignStatus", e.Status)
						device.SetLabel("zone", campaignZoneMapping[e.CampaignID])

						if status, _ := dm.UpdateDevice(stdlib.GetString(APPLICATION_ID, "bobbycar"), device, false); status != http.StatusNoContent {
							log.Trace().Str("vin", e.VIN).Str("campaign", e.CampaignID).Str("executionId", e.CampaignExecutionID).Msg(e.Status)
						} else {
							log.Error().Str("vin", e.VIN).Str("campaign", e.CampaignID).Str("executionId", e.CampaignExecutionID).Int("http", status).Msg("device not updated")
						}
					} else {
						log.Warn().Str("vin", e.VIN).Msg("device not found")
					}
				}
			}
		}
	}
}

// http endpoint setup

func setup() *echo.Echo {
	// create a new router instance
	e := echo.New()

	// add and configure any middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.DefaultCORSConfig))

	// add your own endpoints here
	e.GET("/", api.DefaultEndpoint)
	e.GET("/api/registry/apps/:applicationid/devices/:deviceid", getDeviceEndpoint)

	// done
	return e
}

func shutdown(ctx context.Context, a *apikit.App) error {
	kc.Close() // close kafka listener
	return nil
}

// handler

func getDeviceEndpoint(c echo.Context) error {
	applicationId := c.Param("applicationid")
	if applicationId == "" {
		return api.ErrorResponse(c, http.StatusBadRequest, api.ErrInvalidRoute, "applicationid")
	}
	deviceId := c.Param("deviceid")
	if deviceId == "" {
		return api.ErrorResponse(c, http.StatusBadRequest, api.ErrInvalidRoute, "deviceid")
	}

	status, device := dm.GetDevice(applicationId, deviceId)
	if status != http.StatusOK {
		return api.ErrorResponse(c, http.StatusBadRequest, api.ErrInternalError, "device not found")
	}

	return api.StandardResponse(c, http.StatusOK, device)
}
