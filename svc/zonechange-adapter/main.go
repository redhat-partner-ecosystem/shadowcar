package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	mcache "github.com/OrlovEvgeny/go-mcache"

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

	LOG_LEVEL_DEBUG = "log_level_debug"

	DefaultTTL = time.Minute * 1
)

type (
	ZoneMapping struct {
		Zone      string   `json:"zone,omitempty"`
		Campaigns []string `json:"campaigns,omitempty"`
	}

	ZoneMap []ZoneMapping
)

var (
	zoneToCampaignMapping map[string][]string // zone -> campaigns
	vinToCampaignCache    *mcache.CacheDriver // campaign cache

	mu sync.Mutex
	kc *kafka.Consumer
	cm *ota.CampaignManagerClient
	dm *drogue.DrogueClient
)

func init() {

	// setup logging
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

	// setup other data structures
	vinToCampaignCache = mcache.New()

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

	// setup campaigns and other structs ...

	go func() {
		for {
			refreshCampaignMappings()
			time.Sleep(30 * time.Second) // refesh every 30 sec
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
		var age int64 = stdlib.Now()
		if last, ok := device.GetAnnotation("lastUpdate"); ok {
			lastUpdate, _ := strconv.ParseInt(last, 0, 64)
			age = stdlib.Now() - lastUpdate
		}

		if age > stdlib.GetInt("zone_change_delay", 60) {

			campaign := lookupCampaign(evt.CarID, evt.NextZoneID)

			if campaign != "" {
				if currentCampaign, _ := device.GetAnnotation("campaign"); currentCampaign == campaign {
					log.Info().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Str("campaign", campaign).Int64("age", age).Msg("skipping execute campaign")
				} else {
					log.Info().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Str("campaign", campaign).Int64("age", age).Msg("executing campaign")

					err := cm.ExecuteCampaign(campaign)

					if err != nil {
						log.Error().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Str("campaign", campaign).Err(err).Msg("execute campaign failed")
					} else {
						device.Metadata.Generation++
						device.SetLabel("zone", evt.NextZoneID)
						device.SetAnnotation("lastUpdate", fmt.Sprintf("%d", stdlib.Now()))
						device.SetAnnotation("campaign", campaign)

						dm.UpdateDevice(stdlib.GetString(APPLICATION_ID, "bobbycar"), device, false)
					}
				}
			} else {
				log.Warn().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Msg("no campaign found")
			}
		} else {
			log.Info().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Int64("age", age).Msg("ignoring zone trigger")
		}
	}
}

func lookupCampaign(vin, zone string) string {
	key := fmt.Sprintf("%s-%s", vin, zone)

	if c, ok := vinToCampaignCache.Get(key); ok {
		return c.(*ota.Campaign).CampaignID
	}

	// nothing in the cache, query the campaign manager

	if campaigns, ok := zoneToCampaignMapping[zone]; ok {
		// find the car ...
		for _, campaignId := range campaigns {
			status, campaign := cm.GetCampaign(campaignId)
			if status == http.StatusOK {
				status, group := cm.GetVehicleGroup(campaign.VehicleGroupID)
				if status == http.StatusOK {
					for _, vehicle := range group.VINS {
						if vehicle == vin {
							log.Debug().Str("key", key).Str("campaignId", campaign.CampaignID).Msg("new cache entry")
							vinToCampaignCache.Set(key, &campaign, DefaultTTL)

							return campaign.CampaignID
						}
					}
				} else {
					log.Warn().Str("vehicle_group_id", group.VehicleGroupID).Msg("vehicle_group not found")
				}
			} else {
				log.Warn().Str("campaignId", campaign.CampaignID).Msg("campaign not found")
			}
		}
		return ""
	}

	return ""
}

func lookupVehicle(vin string) *drogue.Device {
	status, device := dm.GetDevice(stdlib.GetString(APPLICATION_ID, "bobbycar"), vin)
	if status != http.StatusOK {
		return nil
	}
	return &device
}

func refreshCampaignMappings() {
	mu.Lock()
	defer mu.Unlock()

	// setup campaigns etc ...
	campaignsJSON := stdlib.GetString("campaigns", "")
	if campaignsJSON != "" {
		var zm ZoneMap
		err := json.Unmarshal([]byte(campaignsJSON), &zm)
		if err != nil {
			log.Err(err).Msg("")
		}

		// new map & values ...
		zoneToCampaignMapping = make(map[string][]string, len(zm))
		for _, z := range zm {
			zoneToCampaignMapping[z.Zone] = z.Campaigns
		}
	} else {
		log.Warn().Msg("no zone-to-campaign mapping found")
	}
}

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
	// TODO: implement your own stuff here
	return nil
}

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
