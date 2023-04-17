package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	// https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/
	AtMostOnce  = 0
	AtLeastOnce = 1
	ExactlyOnce = 2
)

var (
	host     string = "mqtt-endpoint-ws-bobbycar.apps.sdv.sdv.luxoft.com"
	port     string = "443" // 1883
	protocol string = "wss" // tcp
	queue    string = "car"
	username string = "foo-device-gw@bobbycar"
	//username string = "car-simulator-77bc8854cd-jzz42-gw@bobbycar"
	password string = "car123456"
	//password string = "car123456"
	secure bool   = true
	debug  bool   = true
	MyVIN  string = "car-simulator-77bc8854cd-jzz42-0"
)

type (
	Payload struct {
		CarID string `json:"carid"`
		VIN   string `json:"vin"`
	}
)

func main() {
	if debug {
		mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
		mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
		mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
		mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)
	}

	// setup and configuration
	broker := fmt.Sprintf("%s://%s:%s", protocol, host, port)
	fmt.Println("--> Connecting to " + broker)

	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetCleanSession(true)
	opts.SetClientID("test")
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetOnConnectHandler(onConnect)
	opts.SetDefaultPublishHandler(mh)

	//opts.SetKeepAlive(30 * time.Second)
	//opts.SetPingTimeout(30 * time.Second)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	if secure {
		if protocol == "wss" {
			opts.SetTLSConfig(&tls.Config{
				InsecureSkipVerify: true,
			})
		}
	}

	// create a client
	fmt.Println("--> Creating client ... ")
	c := mqtt.NewClient(opts)

	fmt.Println("--> Connecting ... ")
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("--> Error ... ")
		onDisconnect(c)
		log.Fatal(token.Error())
	}
	defer onDisconnect(c)

	fmt.Println("--> Connected ... ")

	/*
		// publish something
		n := 1
		df := Payload{
			CarID: MyVIN,
			VIN:   MyVIN,
		}

		payload, err := json.Marshal(df)
		if err == nil {
			for i := 0; i < n; i++ {
				fmt.Printf("--> Sending message %d/%d ...\n", i+1, n)
				if token := c.Publish(fmt.Sprintf("%s%s", queue, df.VIN), AtMostOnce, false, payload); token.Wait() && token.Error() != nil {
					fmt.Println("--> Error ... ")
					log.Fatal(token.Error())
				}
				fmt.Printf("--> Sent %d ...\n", i+1)
				time.Sleep(10 * time.Second)
			}
		}
	*/
	fmt.Println("--> Done.")
}

var mh mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func onConnect(c mqtt.Client) {
	fmt.Println("*** Connected")
}

func onDisconnect(c mqtt.Client) {
	fmt.Println("*** Disconnecting ... ")
	c.Disconnect(250)
}
