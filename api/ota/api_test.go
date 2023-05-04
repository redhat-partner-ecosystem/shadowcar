package ota

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	//zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func TestNewCampaignManagerClient(t *testing.T) {

	cl, err := NewCampaignManagerClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	assert.NotNil(t, cl.rc.HttpClient)
	assert.NotNil(t, cl.rc.Settings)
	assert.NotNil(t, cl.rc.Settings.Credentials)

	assert.NotEmpty(t, cl.rc.Settings.UserAgent)
	assert.NotEmpty(t, cl.rc.Settings.Endpoint)
	//assert.NotEmpty(t, cl.rc.Settings.Credentials.Token)
}

func TestNewCampaignManagerClientWithOptions(t *testing.T) {

	cl, err := NewCampaignManagerClient(context.TODO(), internal.WithEndpoint("foo.example.com"))
	assert.NotNil(t, cl)
	assert.NoError(t, err)
	assert.NotNil(t, cl.rc.HttpClient)
	assert.NotNil(t, cl.rc.Settings)
	//assert.NotNil(t, cl.rc.Settings.Credentials)

	//assert.Equal(t, "foo", cl.rc.Settings.Credentials.UserID)
	//assert.Equal(t, "bar", cl.rc.Settings.Credentials.Token)

	assert.NotEmpty(t, cl.rc.Settings.Endpoint)
	assert.Equal(t, "foo.example.com", cl.rc.Settings.Endpoint)
}

func TestGetAllCampaigns(t *testing.T) {

	cl, err := NewCampaignManagerClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	status, resp := cl.GetAllCampaigns()
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp)
	assert.Equal(t, http.StatusOK, status)

	//fmt.Println(resp)
}

func TestGetCampaign(t *testing.T) {

	cl, err := NewCampaignManagerClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	status, campaigns := cl.GetAllCampaigns()
	assert.NotNil(t, campaigns)
	assert.NotEmpty(t, campaigns)
	assert.Equal(t, http.StatusOK, status)

	if len(campaigns) > 0 {
		status, resp := cl.GetCampaign(campaigns[0].CampaignID)

		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, campaigns[0].CampaignID, resp.CampaignID)

		//fmt.Println(resp)
	} else {
		assert.Fail(t, "no campaigns found")
	}
}

func TestExecuteCampaign(t *testing.T) {

	cl, err := NewCampaignManagerClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	// check the tick/tock logic
	err = cl.ExecuteCampaign(campaignIdZone1)

	if err == nil {
		err = cl.ExecuteCampaign(campaignIdZone1)
		assert.Error(t, err)
		log.Error().Err(err).Msg("execute campaignIdZone1")

		status, resp := cl.GetCampaignExecution(campaignIdZone1)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp)
		assert.Equal(t, http.StatusOK, status)
	} else {
		log.Error().Err(err).Msg("execute campaignIdZone1")

		err = cl.ExecuteCampaign(campaignIdZone2)
		assert.NoError(t, err)

		status, resp := cl.GetCampaignExecution(campaignIdZone2)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp)
		assert.Equal(t, http.StatusOK, status)
	}
}

func TestGetVehicleGroup(t *testing.T) {

	cl, err := NewCampaignManagerClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	status, campaigns := cl.GetAllCampaigns()
	assert.NotNil(t, campaigns)
	assert.NotEmpty(t, campaigns)
	assert.Equal(t, http.StatusOK, status)

	if len(campaigns) > 0 {
		status, resp := cl.GetVehicleGroup(campaigns[0].VehicleGroupID)

		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, campaigns[0].VehicleGroupID, resp.VehicleGroupID)

		fmt.Println(resp)
	} else {
		assert.Fail(t, "no campaigns found")
	}
}
