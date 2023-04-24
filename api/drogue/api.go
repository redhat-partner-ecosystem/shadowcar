package drogue

import (
	"fmt"
	"net/http"

	"github.com/txsvc/apikit/config"
	"github.com/txsvc/apikit/logger"
	"github.com/txsvc/apikit/settings"
	"github.com/txsvc/stdlib/v2"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
)

const (
	DrogueHttpEndpoint = "DROGUE_HTTP_ENDPOINT"

	DrogueClientID     = "DROGUE_CLIENT_ID"
	DrogueClientSecret = "DROGUE_CLIENT_SECRET"
	DrogueAccessToken  = "DROGUE_ACCESS_TOKEN"

	DrogueApiAgent = "shadowcar/drogue"
)

type (
	Client struct {
		rc internal.RestClient
	}
)

func NewClient(logger logger.Logger) (*Client, error) {

	httpClient := internal.NewTransport(logger, http.DefaultTransport)
	ds := &settings.DialSettings{
		Endpoint:    stdlib.GetString(DrogueHttpEndpoint, ""),
		UserAgent:   DrogueApiAgent,
		Credentials: credentials(),
	}

	if ds.Endpoint == "" {
		return nil, fmt.Errorf("missing DROGUE_HTTP_ENDPOINT")
	}

	if ds.Credentials.UserID != "" && ds.Credentials.Token == "" {
		return nil, fmt.Errorf("missing DROGUE_CLIENT_SECRET")
	}

	return &Client{
		rc: internal.RestClient{
			HttpClient: httpClient,
			Settings:   ds,
			Logger:     logger,
			Trace:      stdlib.GetString(config.ForceTraceENV, ""),
		},
	}, nil
}

func credentials() *settings.Credentials {
	c := &settings.Credentials{
		Token: stdlib.GetString(DrogueAccessToken, ""),
	}
	if c.Token == "" {
		c.UserID = stdlib.GetString(DrogueClientID, "")
		c.Token = stdlib.GetString(DrogueClientSecret, "")

	}
	return c
}

func (c *Client) GetAccessToken() (int, Tokens) {
	var resp Tokens

	status, _ := c.rc.GET("/api/tokens/v1alpha1", &resp)
	if status != http.StatusOK {
		return status, nil
	}

	return status, resp
}

func (c *Client) GetAllDevices(application string) (int, Devices) {
	var resp Devices

	status, _ := c.rc.GET(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices", application), &resp)
	if status != http.StatusOK {
		return status, nil
	}

	return status, resp
}

func (c *Client) GetDevice(application, name string) (int, Device) {
	var resp Device

	status, _ := c.rc.GET(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices/%s", application, name), &resp)
	if status != http.StatusOK {
		return status, Device{}
	}
	return status, resp
}

func (c *Client) CreateDevice(application string, device *Device) (int, Device) {
	status, _ := c.rc.POST(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices", application), device, nil)
	if status != http.StatusCreated {
		return status, Device{}
	}

	status, newDevice := c.GetDevice(application, device.Metadata.Name)
	if status == http.StatusOK {
		return http.StatusCreated, newDevice
	}

	return status, Device{}
}

func (c *Client) RegisterDevice(application, name, user, password string) (int, Device) {
	req := Device{
		Metadata: &ScopedMetadata{
			Name:        name,
			Application: application,
		},
	}

	if user != "" || password != "" {
		req.Spec = &DeviceSpec{}

	}

	if user != "" && password != "" {
		req.Spec.Authentication = &DeviceCredentialStruct{
			User: &UserStruct{
				Username: user,
				Password: password,
			},
		}
	}
	if user == "" && password != "" {
		req.Spec = &DeviceSpec{}
		req.Spec.Authentication = &DeviceCredentialStruct{
			Pass: password,
		}
	}

	status, _ := c.rc.POST(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices", application), &req, nil)
	if status != http.StatusCreated {
		return status, Device{}
	}

	status, device := c.GetDevice(application, name)
	if status == http.StatusOK {
		return http.StatusCreated, device
	}

	return status, Device{}
}

func (c *Client) DeleteDevice(application, name string) int {
	status, _ := c.rc.DELETE(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices/%s", application, name), nil, nil)
	return status
}
