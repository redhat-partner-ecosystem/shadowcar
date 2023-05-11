package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
)

const (
	zone1           = "zone-south"
	campaignIdZone1 = "00000000-0000-0000-0000-aaaaaaaaaaaa"
	zone2           = "zone-north"
	campaignIdZone2 = "00000000-0000-0000-0000-bbbbbbbbbbbb"

	vin            = "WBAFR9C59BC270614"
	VehicleGroupId = "70bd4efc-5c69-4dcc-a7f7-3b126bfe5eab"

	campaignId = "00000000-0000-0000-0000-aaaaaaaaaaaa"
)

func init() {
	//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//zerolog.SetGlobalLevel(zerolog.TraceLevel)
	//zerolog.SetGlobalLevel(zerolog.DebugLevel)
	//zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

/*
func TestInit(t *testing.T) {
	assert.NotNil(t, vinToCampaignCache)
	assert.NotNil(t, cm)
	assert.NotNil(t, kc)

	assert.NotEmpty(t, zoneToCampaignMapping)
}
*/

func TestLogLevel(t *testing.T) {
	//debugEnabled := internal.GetBool(LOG_LEVEL_DEBUG, false)
	//assert.True(t, debugEnabled)

	//log.Trace().Enabled()
}

/*
func _TestLookupCampaign(t *testing.T) {

	campaign := lookupCampaign(vin, zone1)
	assert.Equal(t, campaignIdZone1, campaign)

	campaign = lookupCampaign(vin, zone2)
	assert.Equal(t, campaignIdZone2, campaign)

}

func _TestLookupCampaignWithCache(t *testing.T) {

	campaign := lookupCampaign(vin, zone1)
	assert.Equal(t, campaignIdZone1, campaign)

	campaign = lookupCampaign(vin, zone2)
	assert.Equal(t, campaignIdZone2, campaign)

}

func _TestLookupDevice(t *testing.T) {
	device := lookupVehicle(vin)
	assert.NotNil(t, device)
}

func _TestUpdateDevice(t *testing.T) {
	device := lookupVehicle(vin)
	assert.NotNil(t, device)
}
*/

func _TestRefreshVehicleCampaignStatus(t *testing.T) {
	refreshVehicleCampaignStatus()
}

func _TestZoneChange(t *testing.T) {
	device := lookupVehicle(vin)
	assert.NotNil(t, device)

	evt := internal.ZoneChangeEvent{
		NextZoneID: zone2,
		CarID:      vin,
	}

	handleZoneChange(&evt)
}
