package hanami

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	paho "github.com/eclipse/paho.mqtt.golang"
)

var (
	// ErrNoSecret indicates that no JWT secret was set on the client.
	ErrNoSecret = errors.New("no JWT secret set")

	// ErrNoReplyTo indicates that a message cannot be replied to if it has not ReplyTo value.
	ErrNoReplyTo = errors.New("no $reply field was set on payload")

	// ErrNoSocket indicates that the paho client wasn't initialized.
	ErrNoSocket = errors.New("paho socket not initialized")

	// ErrNotConnected indicates that the client is not connected to a broker.
	ErrNotConnected = errors.New("Not Connected") // this comes from paho mqtt somewhere and should match
)

// Client provides a persistent connection to an MQTT server.
type Client struct {

	// Socket is a paho MQTT client.
	Socket paho.Client

	// Subscriptions is a map of topic subscriptions keyed on topic, then service id.
	Subscriptions Subscriptions

	// PubPrefix is the publish topic prefix.
	PubPrefix string

	// SubPrefix is the subscribe topic prefix.
	SubPrefix string

	// Secret is a byte array used for signing and validating JWT payloads.
	Secret []byte

	// ValueKey is the key used to return a single value in a payload.
	ValueKey string

	// JWTExpiry is the number of seconds a JWT-signed message is valid.
	JWTExpiry time.Duration

	// JWTPrefix will be prefixed to any jwt signed messages.
	JWTPrefix string
}

// New returns an instance of a Hanami client.
func New(host string, o *paho.ClientOptions) *Client {

	if o == nil {
		o = paho.NewClientOptions()
	}

	if len(o.Servers) == 0 {
		o.AddBroker(host)
	}

	return &Client{
		Socket:        paho.NewClient(o),
		Subscriptions: NewSubscriptions(),
		ValueKey:      "v",
		JWTExpiry:     5,
		JWTPrefix:     "jwt:",
	}
}

// Connect to the MQTT server using the provided adapter values.
func (c *Client) Connect() error {
	token := c.Socket.Connect()
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}

	return nil
}

// Publish sends a message to the MQTT broker.
func (c *Client) Publish(topic string, qos byte, retain bool, m interface{}) (b []byte, err error) {
	b, err = c.send(topic, qos, retain, m)
	if err != nil {
		return
	}

	return
}

// PublishSigned signs a message with a JWT signature and sends it to the MQTT broker.
func (c *Client) PublishSigned(topic string, qos byte, retain bool, m interface{}) (b []byte, err error) {
	signed, err := c.signMessage(m)
	if err != nil {
		return
	}

	b, err = c.send(topic, qos, retain, signed)
	if err != nil {
		return
	}

	return
}

// Reply sends a message to the reply topic specified in a payload.
func (c *Client) Reply(in *Payload, qos byte, retain bool, m interface{}) (b []byte, err error) {
	if in.ReplyTo == "" {
		err = ErrNoReplyTo
		return
	}

	pubFunc := c.Publish
	if in.ReplySigned {
		pubFunc = c.PublishSigned
	}

	b, err = pubFunc(in.ReplyTo, qos, retain, m)
	if err != nil {
		return
	}

	return

}

// signMessage converts a message into a payload and signs it with a JWT signature.
func (c *Client) signMessage(m interface{}) (b string, err error) {
	/*if _, ok := m.(Msg); !ok {
		m = map[string]interface{}{
			c.ValueKey: m,
		}
	}*/

	if c.Secret == nil {
		err = ErrNoSecret
		return
	}

	opts := c.Socket.OptionsReader()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":  opts.ClientID(),
		"iat":  time.Now().UTC().Unix(),
		"exp":  time.Now().UTC().Add(time.Second * c.JWTExpiry).Unix(),
		"data": m,
	})

	var p string
	p, err = token.SignedString(c.Secret)
	if err != nil {
		return
	}

	b = c.JWTPrefix + p // Add the jwt: prefix so it can be detected easily on the other end.

	return
}

// send publishes a message to the MQTT broker.
func (c *Client) send(topic string, qos byte, retain bool, m interface{}) (b []byte, err error) {
	if c.Socket == nil {
		err = ErrNoSocket
		return
	}

	if c.PubPrefix != "" {
		topic = c.PubPrefix + "/" + topic
	}

	switch t := m.(type) {
	case string:
		b = []byte(t)
	default:
		b, err = json.Marshal(m)
		if err != nil {
			return
		}
	}

	token := c.Socket.Publish(topic, qos, retain, b)
	token.Wait()
	if token.Error() != nil {
		err = token.Error()
		return
	}

	return
}

// Subscribe opens a new subscription to a topic filter for a sub-client id.
func (c *Client) Subscribe(id string, filter string, qos byte, signed bool, handler Callback) error {

	if c.SubPrefix != "" {
		filter = c.SubPrefix + "/" + filter
	}

	isNew := c.Subscriptions.setByID(id, &Subscription{
		Filter:   filter,
		QOS:      qos,
		Signed:   signed,
		Callback: handler,
	})

	if !isNew {
		return nil
	}

	token := c.Socket.Subscribe(filter, qos, c.inboundHandler)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}

	return nil
}

// inboundHandler builds a paho subscription handler which will call all the callbacks
// for a filter when a match topic arrives.
func (c *Client) inboundHandler(client paho.Client, msg paho.Message) {

	p := &Payload{
		Topic: msg.Topic(),
	}

	b := string(msg.Payload())
	var safe bool
	if len(b) > 5 && b[:4] == c.JWTPrefix { // Decode JWT payload if necessary
		err := p.decodeMessage(c.ValueKey, c.Secret, b[4:]) // Strip jwt: prefix
		if err != nil {
			return
		}
		safe = true
	} else {
		p.parseMessage(c.ValueKey, msg.Payload())
	}

	c.Subscriptions.RLock()
	defer c.Subscriptions.RUnlock()

	for filter, subClients := range c.Subscriptions.internal {
		if elements, ok := MatchTopic(filter, msg.Topic()); ok {
			for _, subClient := range subClients {
				if !subClient.Signed || safe {
					o := p.copy()
					o.Elements = elements
					o.Matched = filter
					go subClient.Callback(o)
				}
			}
		}
	}
}

// Unsubscribe deletes the callbacks for a filter by provided subclient id.
func (c *Client) Unsubscribe(id, filter string) {
	c.Subscriptions.deleteByID(filter, id)
	if c.Subscriptions.isEmpty(filter) {
		c.Socket.Unsubscribe(filter)
	}
}

// UnsubscribeAll deletes all callbacks for all topics by a provided service id.
// If isPrefix is true, then the provided id will be matched using hasPrefix.
func (c *Client) UnsubscribeAll(id string, isPrefix bool) {
	for _, filter := range c.Subscriptions.deleteAllByID(id, isPrefix) {
		if c.Subscriptions.isEmpty(filter) {
			c.Socket.Unsubscribe(filter)
		}
	}
}
