package mqtt

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/listeners"
	cmap "github.com/orcaman/concurrent-map"
)

type Server struct {
	brokerPort int
	users      cmap.ConcurrentMap
}

func New(port int) Server {
	server := Server{
		brokerPort: port,
		users:      cmap.New(),
	}
	return server
}

func (server *Server) ConfigureUser(username string, password string) {
	server.users.Set(username, password)
	log.Printf("Added new user: %s", username)
}

func (server *Server) Start() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	tcp := listeners.NewTCP("tcpbroker", fmt.Sprintf(":%d", server.brokerPort))
	mqttServer := mqtt.New()

	authHandler := createAuth(func(user, passwordByte []byte) bool {
		username := string(user)
		password := string(passwordByte)

		if len(username) == 0 || len(password) == 0 {
			log.Printf("Rejecting connection with empty username or password")
			return false
		}

		if storedPwd, ok := server.users.Get(username); ok {
			if password == storedPwd {
				log.Printf("Authenticated a user: %s", username)
				return true
			} else {
				log.Printf("Password for user %s did not match.", username)
			}
		}
		log.Printf("No user with name: %s", username)
		return false
	})

	err := mqttServer.AddListener(tcp, &listeners.Config{
		Auth: authHandler,
	})
	if err != nil {
		log.Fatal(err)
	}
	go mqttServer.Serve()
	go func() {
		<-sigs
		mqttServer.Close()
		log.Printf("mqtt server closed!")
	}()
}
