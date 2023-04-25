package drogue

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
	DrogueHttpEndpoint = "DROGUE_HTTP_ENDPOINT"

	DrogueClientID     = "DROGUE_CLIENT_ID"
	DrogueClientSecret = "DROGUE_CLIENT_SECRET"
	DrogueAccessToken  = "DROGUE_ACCESS_TOKEN"

	DrogueApiAgent = "shadowcar/drogue"
)

type (
	DrogueClient struct {
		rc internal.RestClient
	}
)

func NewDrogueClient(ctx context.Context, opts ...internal.ClientOption) (*DrogueClient, error) {

	httpClient := internal.NewLoggingTransport(http.DefaultTransport)
	ds := &settings.DialSettings{
		Endpoint:    stdlib.GetString(DrogueHttpEndpoint, ""),
		UserAgent:   DrogueApiAgent,
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
		return nil, fmt.Errorf("missing DROGUE_HTTP_ENDPOINT")
	}

	if ds.Credentials.UserID != "" && ds.Credentials.Token == "" {
		return nil, fmt.Errorf("missing DROGUE_CLIENT_SECRET")
	}

	return &DrogueClient{
		rc: internal.RestClient{
			HttpClient: httpClient,
			Settings:   ds,
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

func (c *DrogueClient) GetAccessToken() (int, Tokens) {
	var resp Tokens

	status, _ := c.rc.GET("/api/tokens/v1alpha1", &resp)
	if status != http.StatusOK {
		return status, nil
	}

	return status, resp
}

func (c *DrogueClient) GetAllDevices(application string) (int, Devices) {
	var resp Devices

	status, _ := c.rc.GET(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices", application), &resp)
	if status != http.StatusOK {
		return status, nil
	}

	return status, resp
}

func (c *DrogueClient) GetDevice(application, name string) (int, Device) {
	var resp Device

	status, _ := c.rc.GET(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices/%s", application, name), &resp)
	if status != http.StatusOK {
		return status, Device{}
	}
	return status, resp
}

func (c *DrogueClient) CreateDevice(application string, device *Device) (int, Device) {
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

func (c *DrogueClient) RegisterDevice(application, name, user, password string) (int, Device) {
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

func (c *DrogueClient) DeleteDevice(application, name string) int {
	status, _ := c.rc.DELETE(fmt.Sprintf("/api/registry/v1alpha1/apps/%s/devices/%s", application, name), nil, nil)
	return status
}
