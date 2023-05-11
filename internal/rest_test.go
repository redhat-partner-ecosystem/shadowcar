package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRestClient(t *testing.T) {

	cl, err := NewRestClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	if cl != nil {
		assert.NotNil(t, cl.HttpClient)
		assert.NotNil(t, cl.Settings)
		assert.NotNil(t, cl.Settings.Credentials)

		assert.NotEmpty(t, cl.Settings.UserAgent)
		assert.NotEmpty(t, cl.Settings.Endpoint)

		assert.NotEmpty(t, cl.Settings.Credentials.UserID)
		assert.NotEmpty(t, cl.Settings.Credentials.Token)
	}
}

func TestNewDrogueRestWithOptions(t *testing.T) {

	cl, err := NewRestClient(context.TODO(), WithEndpoint("foo.example.com"), WithCredentials("foo", "bar"))
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	assert.NotNil(t, cl.HttpClient)
	assert.NotNil(t, cl.Settings)
	assert.NotNil(t, cl.Settings.Credentials)

	assert.Equal(t, "foo", cl.Settings.Credentials.UserID)
	assert.Equal(t, "bar", cl.Settings.Credentials.Token)

	assert.NotEmpty(t, cl.Settings.Endpoint)
	assert.Equal(t, "foo.example.com", cl.Settings.Endpoint)
}
