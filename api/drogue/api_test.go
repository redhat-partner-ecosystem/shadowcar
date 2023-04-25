package drogue

import (
	"context"
	"net/http"
	"testing"

	"github.com/redhat-partner-ecosystem/shadowcar/internal"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

const (
	application = "bobbycar"

	deviceName     = "foo-car"
	deviceUser     = "foo-car-user"
	devicePassword = "foo-car-pass"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	//zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func TestNewDrogueClient(t *testing.T) {

	cl, err := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)
	assert.NoError(t, err)

	assert.NotNil(t, cl.rc.HttpClient)
	assert.NotNil(t, cl.rc.Settings)
	assert.NotNil(t, cl.rc.Settings.Credentials)

	assert.NotEmpty(t, cl.rc.Settings.UserAgent)
	assert.NotEmpty(t, cl.rc.Settings.Endpoint)
	assert.NotEmpty(t, cl.rc.Settings.Credentials.Token)
}

func TestNewDrogueClientWithOptions(t *testing.T) {

	cl, err := NewDrogueClient(context.TODO(), internal.WithEndpoint("foo.example.com"), internal.WithCredentials("foo", "bar"))
	assert.NotNil(t, cl)
	assert.NoError(t, err)
	assert.NotNil(t, cl.rc.HttpClient)
	assert.NotNil(t, cl.rc.Settings)
	assert.NotNil(t, cl.rc.Settings.Credentials)

	assert.Equal(t, "foo", cl.rc.Settings.Credentials.UserID)
	assert.Equal(t, "bar", cl.rc.Settings.Credentials.Token)

	assert.NotEmpty(t, cl.rc.Settings.Endpoint)
	assert.Equal(t, "foo.example.com", cl.rc.Settings.Endpoint)
}

func TestGetAccessToken(t *testing.T) {

	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	status, resp := cl.GetAccessToken()
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp)
	assert.Equal(t, http.StatusOK, status)

	//fmt.Println(resp)
}

func TestGetAllDevices(t *testing.T) {

	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	status, resp := cl.GetAllDevices(application)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp)
	assert.Equal(t, http.StatusOK, status)

	//fmt.Println(resp)
}

func TestGetDevice(t *testing.T) {
	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	status, devices := cl.GetAllDevices(application)
	assert.NotNil(t, devices)
	assert.Equal(t, http.StatusOK, status)

	if len(devices) > 0 {
		status, resp := cl.GetDevice(application, devices[0].Metadata.Name)

		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp)
		assert.Equal(t, http.StatusOK, status)

		//fmt.Println(resp)
	}
}

func TestCreateDevice(t *testing.T) {
	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	device := Device{
		Metadata: &ScopedMetadata{
			Name:        deviceName,
			Application: application,
		},
		Spec: &DeviceSpec{
			Authentication: &DeviceCredentialStruct{
				Pass: devicePassword,
			},
		},
	}

	status, newDevice := cl.CreateDevice(application, &device)
	assert.Equal(t, http.StatusCreated, status)
	assert.NotNil(t, newDevice)
	assert.NotEmpty(t, newDevice)

	// delete the device
	status = cl.DeleteDevice(application, deviceName)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestRegisterAndDeleteDevice(t *testing.T) {
	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	// create the device
	status, device := cl.RegisterDevice(application, deviceName, "", "")
	assert.Equal(t, http.StatusCreated, status)
	assert.NotNil(t, device)
	assert.NotEmpty(t, device)

	// double creation should fail
	status, _ = cl.RegisterDevice(application, deviceName, "", "")
	assert.Equal(t, http.StatusConflict, status)

	// delete the device
	status = cl.DeleteDevice(application, deviceName)
	assert.Equal(t, http.StatusNoContent, status)

	// delete again should fail, not found
	status = cl.DeleteDevice(application, deviceName)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestRegisterDevicePass(t *testing.T) {
	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	// create the device with pass phrase
	status, device := cl.RegisterDevice(application, deviceName, "", devicePassword)
	assert.Equal(t, http.StatusCreated, status)
	assert.NotNil(t, device)
	assert.NotEmpty(t, device)
	if device.Spec != nil {
		assert.Equal(t, devicePassword, device.Spec.Authentication.Pass)
		assert.Nil(t, device.Spec.Authentication.User)
	}

	// delete the device
	status = cl.DeleteDevice(application, deviceName)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestRegisterDeviceUser(t *testing.T) {
	cl, _ := NewDrogueClient(context.TODO())
	assert.NotNil(t, cl)

	// create the device with pass phrase
	status, device := cl.RegisterDevice(application, deviceName, deviceUser, devicePassword)
	assert.Equal(t, http.StatusCreated, status)
	assert.NotNil(t, device)
	assert.NotEmpty(t, device)
	if device.Spec != nil {
		assert.Equal(t, "", device.Spec.Authentication.Pass)
		assert.NotNil(t, device.Spec.Authentication.User)
		assert.Equal(t, deviceUser, device.Spec.Authentication.User.Username)
		assert.Equal(t, devicePassword, device.Spec.Authentication.User.Password)
	}

	// delete the device
	status = cl.DeleteDevice(application, deviceName)
	assert.Equal(t, http.StatusNoContent, status)
}
