package main

import (
	"fmt"
	"testing"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
	"github.com/stretchr/testify/assert"
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

func TestInit(t *testing.T) {
	assert.NotNil(t, vinToCampaignCache)
	assert.NotNil(t, cm)
	assert.NotNil(t, kc)

	assert.NotNil(t, cacheHits)
	assert.NotNil(t, cacheMisses)
	assert.NotNil(t, cacheErrors)

	assert.NotEmpty(t, zoneToCampaignMapping)
}

func TestLogLevel(t *testing.T) {
	debugEnabled := internal.GetBool(LOG_LEVEL_DEBUG, false)
	assert.True(t, debugEnabled)

	//log.Trace().Enabled()
}

func TestLookupCampaign(t *testing.T) {

	campaign := lookupCampaign(vin, zone1)
	assert.Equal(t, campaignIdZone1, campaign)

	campaign = lookupCampaign(vin, zone2)
	assert.Equal(t, campaignIdZone2, campaign)

}

func TestLookupCampaignWithCache(t *testing.T) {

	campaign := lookupCampaign(vin, zone1)
	assert.Equal(t, campaignIdZone1, campaign)
	fmt.Println(cacheMisses)
	fmt.Println(cacheHits)

	campaign = lookupCampaign(vin, zone2)
	assert.Equal(t, campaignIdZone2, campaign)
	fmt.Println(cacheMisses)
	fmt.Println(cacheHits)
}
