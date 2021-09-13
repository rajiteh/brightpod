package hanami

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"

	"github.com/dgrijalva/jwt-go"
)

var (

	// ErrTokenExpired indicates that the JWT token has expired.
	ErrTokenExpired = errors.New("JWT token expired")

	// ErrTokenInvalid indicates that the JWT token could not be parsed.
	ErrTokenInvalid = errors.New("JWT token invalid")

	// ErrTokenMalformed indicates that the JWT payload lacked a data field.
	ErrTokenMalformed = errors.New("no data field in JWT token")
)

const (
	replyToField     = "$reply"
	replySignedField = "$signed"
)

// Msg is a convenience type for a map of interfaces.
type Msg map[string]interface{}

// Callback is a function which will be run when a message is received.
type Callback func(*Payload)

// Payload contains the payload and metadata for a message, inbound and outbound.
type Payload struct {

	// ReplyTo is the topic the client wants a reply on.
	ReplyTo string

	// ReplySigned indicates that the message being sent back should be signed.
	ReplySigned bool

	// Matched is the filter the client matched, including wildcards.
	Matched string

	// Topic is the path the message was received from (or sending to).
	Topic string

	// Elements are the elements that matched any wildcard characters.
	Elements []string

	// Msg is the received data payload.
	Msg Msg

	// Retain indicates that the message should be retained.
	Retain bool

	// Qos indicates that the quality of service for the message.
	QoS byte

	// Error will log a payload error.
	Error error

	// Validated indicates if the message was signed and validated.
	Validated bool
}

// copy creates a new isolated instance of a payload.
func (p *Payload) copy() *Payload {

	copied := &Payload{
		ReplyTo:     p.ReplyTo,
		ReplySigned: p.ReplySigned,
		Matched:     p.Matched,
		Topic:       p.Topic,
		Retain:      p.Retain,
		QoS:         p.QoS,
		Error:       p.Error,
		Validated:   p.Validated,
		Elements:    make([]string, 0, len(p.Elements)),
		Msg:         make(Msg),
	}

	for _, v := range p.Elements {
		copied.Elements = append(copied.Elements, v)
	}
	for k, v := range p.Msg {
		copied.Msg[k] = v
	}

	return copied

}

// parseMessage will parse a JSON byte array into a Msg.
func (p *Payload) parseMessage(vkey string, b []byte) {

	p.Msg = make(Msg)

	if len(b) == 0 {
		return
	}

	// Not all messages are JSON. Determine if the payload is a lone value
	// and if so, wrap it in a value field.
	sv := string(b)
	rv := []rune(sv)
	if len(rv) > 0 && rv[0] != '[' && rv[0] != '{' {
		if onlyLetters(sv) && sv != "true" && sv != "false" {
			sv = `"` + strings.TrimSpace(sv) + `"`
		}

		b = []byte(`{"` + vkey + `":` + sv + `}`)
	}

	err := json.Unmarshal(b, &p.Msg)
	if err != nil {
		p.Error = err
		p.Msg[vkey] = sv
	}

	if _, ok := p.Msg[replyToField]; ok {
		p.ReplyTo = p.Msg[replyToField].(string)
		delete(p.Msg, replyToField)
	}

	if _, ok := p.Msg[replySignedField]; ok {
		p.ReplySigned = p.Msg[replySignedField].(bool)
		delete(p.Msg, replySignedField)
	}

}

// decodeMessage will decode and validate a JWT byte array into a Msg.
func (p *Payload) decodeMessage(vkey string, secret []byte, b string) error {
	token, err := jwt.Parse(b, func(token *jwt.Token) (interface{}, error) { // skip jwt: prefix in b
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		p.Error = ErrTokenInvalid
		return err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if _, ok := claims["data"]; !ok {
			p.Error = ErrTokenMalformed
			return p.Error
		}

		p.Msg = Msg{
			vkey: claims["data"],
		}
	}

	p.Validated = true

	return nil
}

// onlyLetters returns true if any non-letter runes are found in the string.
// Thanks to @user6169399 https://stackoverflow.com/a/38554480
func onlyLetters(str string) bool {
	for _, r := range str {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
