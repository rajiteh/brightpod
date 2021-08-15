package main

import (
	"brightpod/pkg/blower"
	"brightpod/pkg/mqtt"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	server.ConfigureUser("smartfan", "smartfan") // client

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
		var blwr *blower.Blower

		if blowerFromMap, ok := blowers.Get(username); ok {
			blwr = blowerFromMap.(*blower.Blower)
		} else {
			blwr, err = blower.New(username, 6, 25.0, 6000)
			if err != nil {
				log.Printf("Could not create new blower: %s", err)
				return
			}
			log.Printf("Blower with ID %s is now monitored.", username)
		}
		blwrModeInt := int(in.Msg["m"].(float64))
		blwrMode, err := blower.ModeAsString(blwrModeInt)
		if err != nil {
			log.Printf("Could not save the mode: %d", blwrModeInt)
			return
		}
		blwr.SetMode(blwrMode)

		blwr.IsFanRunning = int(in.Msg["s"].(float64)) == 1
		blwr.FirmwareVersion = in.Msg["v"].(float64)
		blwr.FirmwareRevision = in.Msg["rv"].(float64)
		blwr.Fs = in.Msg["fs"].(float64)

		blwr.Touch()

		// Persist the new blower data
		blowers.Set(username, blwr)
		log.Printf("Blower data: %+v", blwr)
	}

	err = client.Subscribe("keepalives", "+/keep_alive", 0, false, keepAliveCallback)

	if err != nil {
		log.Fatal(err)
	}

	err = client.Subscribe("control", "control/+/+", 0, false, func(in *hanami.Payload) {
		blowerID := in.Elements[0]
		command := in.Elements[1]
		value := in.Msg["v"]

		obj, ok := blowers.Get(blowerID)
		if !ok {
			log.Printf("Blower with ID does not exist: %s", blowerID)
			return
		}

		blwr := obj.(*blower.Blower)

		if command == "mode" {
			blwr.SetMode(value.(string))
			publishBlowerStatus(client, blwr)
		} else if command == "power" {
			powerValuePercentage, ok := value.(float64)
			if !ok {
				log.Printf("Could not parse the power value: %s", value)
			}
			powerValue := 12 * (int(powerValuePercentage) / 100)
			blwr.SetFanPower(powerValue)
			publishBlowerStatus(client, blwr)
		} else if command == "max_rpm" {
			rpmValue, ok := value.(float64)
			if !ok {
				log.Printf("Could not parse the power value: %s", value)
			}
			blwr.SetFanRPM(int(rpmValue))
			publishBlowerStatus(client, blwr)
		} else {
			log.Printf("Unknown control commmand: %s", command)
			return
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	<-sigs
	client.UnsubscribeAll("keepalives", false)
	log.Println(aurora.BgGreen("Finished"))
}

func publishBlowerStatus(client *hanami.Client, blwr *blower.Blower) {
	topic := fmt.Sprintf("%s/status", blwr.ID)
	payload := blwr.GenerateStausPayload()
	client.Publish(topic, 0, false, payload)
}
