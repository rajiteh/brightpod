package main

import (
	"brightpod/pkg/mqtt"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/logrusorgru/aurora"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/mochi-co/hanami"

	"github.com/ReneKroon/ttlcache/v2"
)

const MQTT_SERVER_USERNAME = "brightpod"
const MQTT_SERVER_PASSWORD = "brightpod"
const BROKER_PORT = 1883

func main() {

	fmt.Println(aurora.Magenta(fmt.Sprintf("Starting MQTT service on port: %d", BROKER_PORT)))
	server := mqtt.New(BROKER_PORT)
	server.Start()
	server.ConfigureUser(MQTT_SERVER_USERNAME, MQTT_SERVER_PASSWORD)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	blowers := ttlcache.NewCache()

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

		if cached, err := blowers.Get(username); err != ttlcache.ErrNotFound {
			blower = cached.(*Blower)
		} else {
			log.Printf("Blower with ID %s is now active.", username)
		}

		blower.mode = in.Msg["m"].(float64)
		blower.s = in.Msg["s"].(float64)
		blower.firmwareVersion = in.Msg["v"].(float64)
		blower.rv = in.Msg["rv"].(float64)
		blowers.Set(username, blower)
		log.Printf("Blower data: %s", blower)
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
	firmwareVersion float64
	s               float64
	mode            float64
	rv              float64
	// status1         *int     //1
	// status2         *int     //6000
	// status3         *int     //6000
	// status4         *float32 //21.0
	// status5         *int     //0
}
