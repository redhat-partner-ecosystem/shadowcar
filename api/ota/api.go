package ota

import (
	"context"
	"fmt"
	"net/http"

	"github.com/txsvc/apikit/config"
	"github.com/txsvc/stdlib/v2"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
	"github.com/redhat-partner-ecosystem/shadowcar/internal/settings"
)

const (
	CampaignManagerHttpEndpoint = "CAMPAIGN_MANAGER_HTTP_ENDPOINT"

	CampaignManagerClientID     = "CAMPAIGN_MANAGER_CLIENT_ID"
	CampaignManagerClientSecret = "CAMPAIGN_MANAGER_CLIENT_SECRET"
	CampaignManagerAccessToken  = "CAMPAIGN_MANAGER_ACCESS_TOKEN"

	CampaignManagerApiAgent = "shadowcar/campaignmanager"
)

type (
	CampaignManagerClient struct {
		rc internal.RestClient
	}
)

func NewCampaignManagerClient(ctx context.Context, opts ...internal.ClientOption) (*CampaignManagerClient, error) {

	httpClient := internal.NewLoggingTransport(http.DefaultTransport)
	ds := &settings.DialSettings{
		Endpoint:    stdlib.GetString(CampaignManagerHttpEndpoint, ""),
		UserAgent:   CampaignManagerApiAgent,
		Credentials: credentials(),
	}

	// apply options
	if len(opts) > 0 {
		for _, opt := range opts {
			opt.Apply(ds)
		}
	}

	// do some basic validation
	if ds.Endpoint == "" {
		return nil, fmt.Errorf("missing CAMPAIGN_MANAGER_HTTP_ENDPOINT")
	}

	/*
		if ds.Credentials.UserID != "" && ds.Credentials.Token == "" {
			return nil, fmt.Errorf("missing DROGUE_CLIENT_SECRET")
		}
	*/
	return &CampaignManagerClient{
		rc: internal.RestClient{
			HttpClient: httpClient,
			Settings:   ds,
			Trace:      stdlib.GetString(config.ForceTraceENV, ""),
		},
	}, nil
}

func credentials() *settings.Credentials {
	c := &settings.Credentials{
		Token: stdlib.GetString(CampaignManagerAccessToken, ""),
	}
	if c.Token == "" {
		c.UserID = stdlib.GetString(CampaignManagerClientID, "")
		c.Token = stdlib.GetString(CampaignManagerClientSecret, "")

	}
	return c
}

func (c *CampaignManagerClient) GetAllCampaigns() (int, Campaigns) {
	var resp Campaigns

	status, _ := c.rc.GET("/campaign", &resp)
	if status != http.StatusOK {
		return status, nil
	}

	return status, resp
}

func (c *CampaignManagerClient) GetCampaign(campaignId string) (int, Campaign) {
	var resp Campaign

	status, _ := c.rc.GET(fmt.Sprintf("/campaign/%s", campaignId), &resp)
	if status != http.StatusOK {
		return status, Campaign{}
	}

	return status, resp
}

func (c *CampaignManagerClient) GetVehicleGroup(vehicleGroupId string) (int, VehicleGroup) {
	var resp VehicleGroup

	status, _ := c.rc.GET(fmt.Sprintf("/vehicle_group/%s", vehicleGroupId), &resp)
	if status != http.StatusOK {
		return status, VehicleGroup{}
	}

	return status, resp
}
