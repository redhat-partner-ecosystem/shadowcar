package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/redhat-partner-ecosystem/shadowcar/api/drogue"
	"github.com/txsvc/apikit/logger"
)

func main() {

	var application string
	var deviceName string
	var devicePassword string

	flag.StringVar(&application, "application", "bobbycar", "Drogue App")
	flag.StringVar(&deviceName, "name", "foo-device", "Device name")
	flag.StringVar(&devicePassword, "password", "car123456", "Device password")

	cl := drogue.NewClient(logger.NewLogger(os.Stdout, "debug"))

	device := drogue.Device{
		Metadata: &drogue.ScopedMetadata{
			Name:        deviceName,
			Application: application,
		},
		Spec: &drogue.DeviceSpec{
			GatewaySelector: &drogue.GatewaySelectorStruct{
				MatchName: []string{fmt.Sprintf("%s-gw", deviceName)},
			},
		},
	}

	if status, _ := cl.CreateDevice(application, &device); status != http.StatusCreated {
		log.Fatal(fmt.Errorf("can not create device '%s'", device.Metadata.Name))
	}

	gw_device := drogue.Device{
		Metadata: &drogue.ScopedMetadata{
			Name:        fmt.Sprintf("%s-gw", deviceName),
			Application: application,
		},
		Spec: &drogue.DeviceSpec{
			Authentication: &drogue.DeviceCredentialStruct{
				Pass: devicePassword,
			},
		},
	}

	if status, _ := cl.CreateDevice(application, &gw_device); status != http.StatusCreated {
		log.Fatal(fmt.Errorf("can not create device '%s'", gw_device.Metadata.Name))
	}

}
