package cmd

import (
	"brightpod/cmd/util"
	"brightpod/pkg/client"
	"brightpod/pkg/mqtt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

type ConfigArguments struct {
	mqttServer      bool
	mqttServerUsers []string
	mqttUsername    string
	mqttPassword    string
	mqttHost        string
}

const (
	brokerPort = 1883
)

func NewRootCommand() *cobra.Command {

	configArgs := ConfigArguments{
		mqttServer:      false,
		mqttServerUsers: []string{},
		mqttUsername:    "",
		mqttPassword:    "",
		mqttHost:        "",
	}

	// Define our command
	rootCmd := &cobra.Command{
		Use:   "brightpod",
		Short: "Smartfan controller",
		Long:  `Connects or runs a mqtt server that allows a connected smart fan to be controlled via HomeAssistant`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			return util.InitializeConfig(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("%+v", configArgs)
			runProgram(&configArgs)
		},
	}

	// mqtt server
	rootCmd.PersistentFlags().BoolVar(&configArgs.mqttServer,
		"mqtt-run-server", false, "Runs the built-in mqtt server.")
	rootCmd.PersistentFlags().StringSliceVar(&configArgs.mqttServerUsers,
		"mqtt-server-users", []string{}, "Users that can access the built-in mqtt server.")

	// app config
	rootCmd.PersistentFlags().StringVar(&configArgs.mqttUsername,
		"mqtt-username", "", "Defines the username to connect to the mqtt instance")
	rootCmd.PersistentFlags().StringVar(&configArgs.mqttPassword,
		"mqtt-password", "", "Defines the password to connect to the mqtt instance")
	rootCmd.PersistentFlags().StringVar(&configArgs.mqttHost,
		"mqtt-host", "", "Defines the password to connect to the mqtt instance")

	return rootCmd
}

func runProgram(config *ConfigArguments) {
	if config.mqttServer {
		log.Printf("Starting MQTT service on port: %d", brokerPort)
		server := mqtt.New(brokerPort)
		server.Start()

		server.ConfigureUser(config.mqttUsername, config.mqttPassword)
		for _, userPassword := range config.mqttServerUsers {
			userPwdSplit := strings.SplitN(userPassword, ":", 2)
			if len(userPwdSplit) != 2 {
				log.Fatalf("Cannot parse user password credential: %s", userPassword)
			}
			server.ConfigureUser(userPwdSplit[0], userPwdSplit[1])
		}
	}

	client.Start(config.mqttUsername, config.mqttPassword, config.mqttHost)
}
