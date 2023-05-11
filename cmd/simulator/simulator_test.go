package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/txsvc/stdlib/v2"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
)

func TestNewRestClient(t *testing.T) {

	cl, err := internal.NewRestClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)
	assert.NotEmpty(t, cl.Settings.Credentials.UserID)
	assert.NotEmpty(t, cl.Settings.Credentials.Token)
}

func TestPostPayload(t *testing.T) {

	cl, err := internal.NewRestClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)
	assert.NotEmpty(t, cl.Settings.Credentials.UserID)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // test server certificate is not trusted.
			},
		},
	}
	assert.NotNil(t, client)
	cl.SetClient(client)

	coord := Coordinates{
		VIN:       stdlib.GetString("VIN", VIN),
		EventTime: stdlib.Now(),
		Lat:       39.78876,
		Long:      -86.23759,
	}

	status, err := cl.POST("/v1/car", coord, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, status)
}
