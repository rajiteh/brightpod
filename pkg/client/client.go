package client

import (
	"brightpod/pkg/blower"
	"brightpod/pkg/client/protocol"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/logrusorgru/aurora"
	"github.com/mochi-co/hanami"

	cmap "github.com/orcaman/concurrent-map"

	paho "github.com/eclipse/paho.mqtt.golang"
)

var (
	blowers = cmap.New()
	client  *hanami.Client
)

func Start(clientUsername, clientPassword, mqttServer string) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	options := paho.NewClientOptions()
	options.Username = clientUsername
	options.Password = clientPassword

	client = hanami.New(mqttServer, options)

	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

	err = client.Subscribe("keepalives", "+/keep_alive", 0, false, handleKeepAlive)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Subscribe("control", "control/+/+", 0, false, handleControl)
	if err != nil {
		log.Fatal(err)
	}

	<-sigs
	client.UnsubscribeAll("keepalives", false)
	client.UnsubscribeAll("control", false)
	log.Println(aurora.BgGreen("Finished"))
}

func handleKeepAlive(in *hanami.Payload) {
	var blwr *blower.Blower

	username := in.Elements[0]
	kaMsg, err := protocol.ParseKeepAlive(in.Msg)
	if err != nil {
		log.Printf("Could not parse keepalive: %s", err.Error())
		return
	}

	if blowerFromMap, ok := blowers.Get(username); ok {
		blwr = blowerFromMap.(*blower.Blower)
	} else {
		blwr, err = blower.New(username, 6, 25.0, 6000, kaMsg.V, kaMsg.RV, kaMsg.FS)
		if err != nil {
			log.Printf("Could not create new blower: %s", err)
			return
		}
		log.Printf("Blower with ID %s is now monitored.", username)
	}
	if err := blwr.SetModeFromInt(kaMsg.M); err != nil {
		log.Printf("Could not set mode to: %s", err.Error())
		return
	}

	blwr.IsFanRunning = kaMsg.S == 1

	// Refresh last seen time
	blwr.UpdateLastContact()

	// Persist the new blower data
	blowers.Set(username, blwr)
	log.Printf("Blower data: %+v", blwr)
}

func publishBlowerStatus(client *hanami.Client, blwr *blower.Blower) {
	topic := fmt.Sprintf("%s/status", blwr.ID())
	// topic := ""
	payload := blwr.GenerateStausPayload()
	client.Publish(topic, 0, false, payload)
}

func handleControl(in *hanami.Payload) {
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
		blwr.SetModeFromString(value.(string))
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
		// rpmValue, ok := value.(float64)
		// if !ok {
		// 	log.Printf("Could not parse the power value: %s", value)
		// }
		// blwr.SetFanRPM(int(rpmValue))
		publishBlowerStatus(client, blwr)
	} else {
		log.Printf("Unknown control commmand: %s", command)
		return
	}
}
