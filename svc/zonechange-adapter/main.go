package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redhat-partner-ecosystem/shadowcar/api/ota"
	"github.com/redhat-partner-ecosystem/shadowcar/internal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	mcache "github.com/OrlovEvgeny/go-mcache"

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

	cacheHits   prometheus.Counter // metrics
	cacheMisses prometheus.Counter
	cacheErrors prometheus.Counter

	mu sync.Mutex
	kc *kafka.Consumer
	cm *ota.CampaignManagerClient
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

	// setup other data structures
	vinToCampaignCache = mcache.New()

	// campaign manager client
	cm, err = ota.NewCampaignManagerClient(context.TODO())
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

	// setup prometheus

	cacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "campaign_cache_hits",
		Help: "The number of cache hits looking up a campaign",
	})
	cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "campaign_cache_misses",
		Help: "The number of cache misses looking up a campaign",
	})
	cacheErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "campaign_cache_errors",
		Help: "The number of cache errors",
	})

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
	if evt.NextZoneID != "" {
		// only do sth in case a car enters a zone
		campaign := lookupCampaign(evt.CarID, evt.NextZoneID)

		if campaign != "" {
			log.Info().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Str("campaign", campaign).Msg("execute campaign")

			err := cm.ExecuteCampaign(campaign)
			if err != nil {
				log.Error().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Str("campaign", campaign).Err(err).Msg("execute campaign failed")
			}
		} else {
			log.Warn().Str("vin", evt.CarID).Str("zone", evt.NextZoneID).Msg("no campaign found")
		}
	}
}

func lookupCampaign(vin, zone string) string {
	key := fmt.Sprintf("%s-%s", vin, zone)

	if c, ok := vinToCampaignCache.Get(key); ok {
		cacheHits.Inc()
		return c.(*ota.Campaign).CampaignID
	}

	// nothing in the cache, query the campaign manager
	cacheMisses.Inc()

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

	cacheErrors.Inc()
	return ""
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
