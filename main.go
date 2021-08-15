package main

import (
	"brightpod/pkg/mqtt"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logrusorgru/aurora"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/mochi-co/hanami"
	cmap "github.com/orcaman/concurrent-map"
)

const MQTT_SERVER_USERNAME = "brightpod"
const MQTT_SERVER_PASSWORD = "brightpod"
const BROKER_PORT = 1883

func main() {

	fmt.Println(aurora.Magenta(fmt.Sprintf("Starting MQTT service on port: %d", BROKER_PORT)))
	server := mqtt.New(BROKER_PORT)
	server.Start()

	// User configuration
	server.ConfigureUser(MQTT_SERVER_USERNAME, MQTT_SERVER_PASSWORD)
	server.ConfigureUser("mqttexplorer", "debug")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	blowers := cmap.New()

	options := paho.NewClientOptions()
	options.Username = MQTT_SERVER_USERNAME
	options.Password = MQTT_SERVER_PASSWORD

	client := hanami.New(fmt.Sprintf("tcp://localhost:%d", BROKER_PORT), options)
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

	keepAliveCallback := func(in *hanami.Payload) {
		log.Printf("RECV KA: %+v\n", in)
		username := in.Elements[0]
		var blower *Blower

		if blowerFromMap, ok := blowers.Get(username); ok {
			blower = blowerFromMap.(*Blower)
		} else {
			blower = &Blower{
				fanPower:    6,
				temperature: 25.0,
				rpm:         6000, // default from their backend, no idea?
			}
			blowers.Set(username, blower)
			log.Printf("Blower with ID %s is now monitored.", username)
		}

		blower.mode = in.Msg["m"].(float64)
		blower.isFanRunning = in.Msg["s"].(float64)
		blower.firmwareVersion = in.Msg["v"].(float64)
		blower.firmwareRevision = in.Msg["rv"].(float64)
		blower.fs = in.Msg["fs"].(float64)
		blower.lastKeepAlive = time.Now()
		log.Printf("Blower data: mode=%f", blower.mode)
	}

	err = client.Subscribe("keepalives", "+/keep_alive", 0, false, keepAliveCallback)

	if err != nil {
		log.Fatal(err)
	}

	<-sigs
	client.UnsubscribeAll("keepalives", false)
	log.Println(aurora.BgGreen("  Finished  "))
}

type Blower struct {
	id               string
	firmwareVersion  float64
	firmwareRevision float64
	isFanRunning     float64
	mode             float64
	fs               float64
	fanPower         int
	rpm              int
	temperature      float64
	lastKeepAlive    time.Time
}
