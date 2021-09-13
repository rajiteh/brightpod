
<p align="center"><img src="img/hanami-splash.svg" alt="mochi-co/hanami MQTT client"></p>

<p align="center">
  
[![Build Status](https://travis-ci.com/mochi-co/hanami.svg?token=59nqixhtefy2iQRwsPcu&branch=master)](https://travis-ci.com/mochi-co/hanami)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/mochi-co/hanami/issues)
[![codecov](https://codecov.io/gh/mochi-co/hanami/branch/master/graph/badge.svg?token=6vBUgYVaVB)](https://codecov.io/gh/mochi-co/hanami)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/b48e17f87cee4221b60a45c02d49148c)](https://www.codacy.com/app/mochi-co/hanami?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=mochi-co/hanami&amp;utm_campaign=Badge_Grade)
[![GoDoc](https://godoc.org/github.com/mochi-co/hanami?status.svg)](https://godoc.org/github.com/mochi-co/hanami)

</p>

## What is Hanami?
Hanami is a wrapper for the [Go Paho MQTT Client](https://github.com/eclipse/paho.mqtt.golang) which provides a few convenience functions that are designed to make life a little easier. It was developed to alleviate a few pain points with the Paho client such as adding multiple callbacks per topic, and adds JWT message signing and Reply-To helpers for implementing request-response patterns. 


## Main Features
* Add multiple callback functions per topic, isolated by sub-client id.
* Inbound and outbound JWT message signing to ensure only payloads from trusted sources are processed.
* Reply-To helpers for lazy request-response pattern implementation. 

## Quick Start
```go
import "github.com/mochi-co/hanami"
```

Hanami wraps the standard `paho.Client` and takes a `host` address and a `paho.ClientOptions`. Multiple hosts can be added by setting them in the options directly, in which case the `host` string will be ignored in favour of the options value.

```go
import (
	"log"
	"github.com/mochi-co/hanami"
)

func main() {

	// The hanami client takes standard paho options.
	options := paho.NewClientOptions()

	// Create the new hanami client with the broker address and the paho options.
	client := hanami.New("tcp://iot.eclipse.org:1883", options)

	// Connecting the client is the same as connecting the paho client, 
	// minus the boilerplate token code. It is non-blocking.
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

}	

```

## Examples
Examples can be found in [examples](https://github.com/mochi-co/hanami/tree/master/example)

## Using Hanami [![GoDoc](https://godoc.org/github.com/mochi-co/hanami?status.svg)](https://godoc.org/github.com/mochi-co/hanami)
> ~ The following guide assumes existing knowledge of MQTT and the Go paho client.

* [New - Create a new Hanami client](#hanaminewhost-string-o-pahoclientoptions-hanamiclient)
* [Connect - Connect to a broker](#clientconnect-error)
* [Publish - Publish a message to a broker](#clientpublishtopic-string-qos-byte-retain-bool-m-interface-b-byte-err-error)
* [PublishSigned - Publish a JWT-signed message to a broker](#clientpublishsignedtopic-string-qos-byte-retain-bool-m-interface-b-byte-err-error)
* [Reply - Reply to an incoming message on a reply-to topic](#clientreplyin-payload-qos-byte-retain-bool-m-interface-b-byte-err-error)
* [Subscribe - Subscribe a sub-client to a topic filter](#clientsubscribeid-string-filter-string-qos-byte-signed-bool-handler-hanamicallback-error)
* [Unsubscribe - Unsubscribe a sub-client from a topic filter](#clientunsubscribeid-filter-string)
* [UnsubscribeAll - Remove all subscriptions for a specific sub-client](#clientunsubscribeallid-string-isprefix-bool)


##### `hanami.New(host string, o *paho.ClientOptions) *hanami.Client`
Create a new Hanami client. `host` takes the address of the MQTT broker to connect to (eg. `"tcp://iot.eclipse.org:1883"`) and a `paho.ClientOptions` containing [configuration parameters](https://github.com/eclipse/paho.mqtt.golang/blob/master/options.go) for the internal `paho` client. Multiple broker addresses can be assigned by setting them each using `options.AddBroker(host)` before calling `hanami.Connect`. A new Hanami client is returned.

Various Hanami specific parameters can be configured by directly setting values on the client:
```go
client.Secret = []byte{"my-jwt-secret"} // Set the JWT signing secret.
client.JWTExpiry = 10 // Number of seconds a signed message is valid.
client.JWTPrefix = "jwt:" // An indicator string to prefix JWT payloads.
client.PubPrefix = "hanami/out" // Add a prefix to all publishing topics.
client.SubPrefix = "hanami/in" // Add a prefix to all subscribing topics.

```

The `PubPrefix` and `SubPrefix` values will be prepended to any provided topic values and joined with a `/`. For example, calling `Subscribe` or `Publish` with a topic of `bar/baz` will normally subscribe to `bar/baz`. If the SubPrefix of `foo` is set, then the actual subscribed topic will be `foo/bar/baz`.


```go
options := paho.NewClientOptions() // Set various standard client options...
options.SetClientID("mqtt-client-id")
options.SetUsername("user")
options.SetPassword("password")

client := hanami.New("tcp://iot.eclipse.org:1883", options)
```

```go
options := paho.NewClientOptions()
options.SetClientID("mqtt-client-id")
options.AddBroker("tcp://iot.eclipse.org:1883") // Or add multiple broker addresses
options.AddBroker("tcp://test.mosquitto.org:1883")

client := hanami.New("", options) // host is unneeded when setting brokers manually.
```

##### `client.Connect() error`  
Connect to an MQTT broker. The Client will connect to the MQTT broker specified in when calling `hanami.New`. 

```go
err := client.Connect()
```

-----

##### `client.Publish(topic string, qos byte, retain bool, m interface{}) (b []byte, err error)`  
Publish a message to the connected broker. Takes the same parameters as `paho.Publish`, and returns the `b` byte array which was sent to the broker if successful. `m` values will be marshalled to `json`, unless they are strings; which are converted directly into a byte array to avoid unnecessarily wrapping the value in quote marks. If `client.PubPrefix` has been set, the topic will be prepended with the value. `m` may be any value: int, struct, map, string, bool, etc.

```go
b, err := client.Publish("hanami/example/map", 1, false, map[string]interface{}{
	"v": "this is my value",
})
// b == [123 34 118 34 58 34 116 104 105 115 32 105 115 32 109 121 32 118 97 108 117 101 34 125]
// string(b) == {"v":"this is my value"}
```

-----

##### `client.PublishSigned(topic string, qos byte, retain bool, m interface{}) (b []byte, err error)`  
Publish a JWT signed messaged to the connected broker. `client.Secret` **MUST** be set in order to use any of the signing features. Operates exactly the same as `client.Publish`, except the returned byte array will always be a JWT encoded token prefixed with the client.JWTPrefix indicator (by default this is `jwt:`). The prefix is used to determine if the payload is signed so similarly formatted messages can be sent to the broker from other clients, and they will be processed as expected.

```go
client.Secret = []byte("hanami-test") // A Secret MUST be set.
_, err = client.PublishSigned("hanami/example/signed", 1, false, "a signed test")
// b == [106 119 116 58 101 121 74 104 98 71 99 105 79 105 74 73 85 122 73 49 78 105 73 115 73 110 82 53 99 67 73 54 73 107 112 88 86 67 74 57 46 101 121 74 107 89 88 82 104 73 106 111 105 89 83 66 122 97 87 100 117 90 87 81 103 100 71 86 122 100 67 73 115 73 109 86 52 99 67 73 54 77 84 85 50 78 68 81 51 77 122 85 120 78 105 119 105 97 87 70 48 73 106 111 120 78 84 89 48 78 68 99 122 78 84 69 120 76 67 74 112 99 51 77 105 79 105 73 105 102 81 46 80 84 73 69 98 55 65 70 69 122 72 119 80 97 82 48 103 112 66 70 84 109 82 104 97 66 78 119 106 81 56 81 121 85 104 54 104 78 51 78 85 104 85]
// string(b) == jwt:eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjoiYSBzaWduZWQgdGVzdCIsImV4cCI6MTU2NDQ3MzUxNiwiaWF0IjoxNTY0NDczNTExLCJpc3MiOiIifQ.PTIEb7AFEzHwPaR0gpBFTmRhaBNwjQ8QyUh6hN3NUhU
```

-----

##### `client.Reply(in *Payload, qos byte, retain bool, m interface{}) (b []byte, err error)` 
Reply to a message which contained a `$reply` field in the original payload. In order for a received message to be replyable, the payload must consist of JSON map containing the fields of at least `$reply`. The message may also indicate that the reply should be automatically signed, by specifying the `$signed` field. The `$reply` and `$signed` meta fields will be stripped from the final payload.

```go
// Originating message sent to broker.
b, err := client.Publish("hanami/example/a/set", 1, false, map[string]interface{}{
	"v": "this is my value",
	"$reply": "hanami/example/a", // request that replies be sent to this topic instead.
})

// Originating message payload (`in *Payload`) as received by Hanami. 
var in *Payload = &Payload{ 
	Msg: [116 104 105 115 32 105 115 32 109 121 32 118 97 108 117 101],
	ReplyTo: "hanami/example/a",
	ReplySigned: false,
	// ... Other fields omitted for readability.
}


// Reply to the originating message using client.Reply...
b2, err := client.Reply(in, 0, false, "this is my reply value")
```

In the above example, a message is received by Hanami containing the special meta field of `$reply`, requesting that replies be sent to `"hanami/example/a"`. The payload received can hten be passed directly to the `client.Reply` method, which will handle automatically signing based on `ReplySigned`, and then send to whichever topic is specified in `ReplyTo`. 

> ! Note! Reply implements `PubPrefix`. Topics being sent as a reply will have PubPrefix appended.

-----

##### `client.Subscribe(id string, filter string, qos byte, signed bool, handler hanami.Callback) error`  
Subscribe to a topic filter. In `hanami`, subscriptions are virtualized using _sub-clients,_ so you can have multiple callbacks per topic filter by specifying unique `id` values. The `signed` parameter indicates that the filter expects the payload to be signed with a valid JWT token matching `client.Secret`. If `signed` is true, and the payload is not signed or is invalid/expired, the payload will be ignored for that handler. It is possible a have two subscriptions (`a` and `b`), where `a` expects a signed payload and b does not. On receiving a non-signed payload, only `b` will process the payload.

```go
// Create a callback handler that receives a `*hanami.Payload`
cb := func(in *hanami.Payload) {
	log.Printf("RECV: %+v\n", in)
}

// Subscribe to a topic filter and handle all matched incoming messages with our `cb` handler.
err := client.Subscribe("a", "hanami/example/+", 0, false, cb)

// Multiple callbacks can be added to the same filter by changing the unique sub-client id.
// Handlers may also be passed in directly.
err := client.Subscribe("b", "hanami/example/+", 0, false, func(in *hanami.Payload) {
	log.Printf("This is another callback that will be called also, for %s", in.Topic)
})

// Subscribe to a signed topic by setting signed to true.
err := client.Subscribe("a", "hanami/example/signed", 0, true, func(in *hanami.Payload) {
	// The payload will be automatically decoded if the incoming token is valid. 
	// Any non-JWT messages will be dropped by this sub-client.
})
```

-----

##### `client.Unsubscribe(id, filter string)`  
Unsubscribe removes a topic filter subscription by sub-client id. `hanami` maintains one subscription per filter _(not_ sub-client), so if the sub-client is the last or only callback remaining for the filter, the filter will be unsubcribed from the `paho client.`
```go
client.Unsubscribe("a", "hanami/example")
```

-----

##### `client.UnsubscribeAll(id string, isPrefix bool)`  
UnsubscribeAll removes _all_ callbacks for a specific sub-client id. If `isPrefix` is true, the id will be treated as a prefix and will unsubscribe any matching ids.

```go
// Assuming the following subscribed filters:
// my/filter/hello - subclients: ["a","b","c"]
// another/filters - subclients: ["a"]
// another/filter/stuff - subclients: ["b","c"]

client.UnsubscribeAll("a")

// Subscribed filters then becomes:
// my/filter/hello - subclients: ["b","c"]
// another/filter/stuff - subclients: ["b","c"]
```

-----

## Why 'Hanami'?
This project was born out of a need while developing [Sakura](https://github.com/mochi-co/sakura), a lightweight Go MQTT broker for embedding in small iot, smarthome, and other non-enterprise projects. The name Sakura (桜) was chosen following the metaphor that the messages running through the broker was akin to the cherry blossoms that fall every year - each unique, but countless. Since this client was originally designed to work with _Sakura,_ the metaphor was naturally extended to _Hanami,_ which is to watch the cherry blossoms.

## Contributions
Contributions to Hanami are both welcome and encouraged! Open an [issue](https://github.com/mochi-co/hanami/issues) to report a bug or make a feature request. Participation in the project is governed by our [code of conduct](https://github.com/mochi-co/hanami/blob/master/CODE_OF_CONDUCT.md). 