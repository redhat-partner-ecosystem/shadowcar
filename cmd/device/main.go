package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/redhat-partner-ecosystem/shadowcar/api/drogue"
)

func main() {

	var application string
	var deviceName string
	var devicePassword string

	flag.StringVar(&application, "application", "bobbycar", "Drogue App")
	flag.StringVar(&deviceName, "name", "WP0AA2991YS620631", "Device name")
	flag.StringVar(&devicePassword, "password", "car123456", "Device password")

	cl, err := drogue.NewDrogueClient(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	gatewayDeviceName := fmt.Sprintf("%s-gw", deviceName)

	gw_device := drogue.Device{
		Metadata: &drogue.ScopedMetadata{
			Name:        gatewayDeviceName,
			Application: application,
		},
		Spec: &drogue.DeviceSpec{
			Authentication: &drogue.DeviceCredentialStruct{
				Pass: devicePassword,
			},
		},
	}

	if status, _ := cl.CreateDevice(application, &gw_device); status != http.StatusCreated {
		log.Fatal(fmt.Errorf("can not create gateway device '%s'", gw_device.Metadata.Name))
	}

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
		cl.DeleteDevice(application, gatewayDeviceName) // try to delete the gateway, ignore the outcome
		log.Fatal(fmt.Errorf("can not create device '%s'", device.Metadata.Name))
	}

}
