package listeners

import (
	"net"
	"sync"

	"github.com/mochi-co/mqtt/server/listeners/auth"
	"github.com/mochi-co/mqtt/server/system"
)

// Config contains configuration values for a listener.
type Config struct {
	Auth auth.Controller // an authentication controller containing auth and ACL logic.
	TLS  *TLS            // the TLS certficates and settings for the connection.
}

// TLS contains the TLS certificates and settings for the listener connection.
type TLS struct {
	Certificate []byte // the body of a public certificate.
	PrivateKey  []byte // the body of a private key.
}

// EstablishFunc is a callback function for establishing new clients.
type EstablishFunc func(id string, c net.Conn, ac auth.Controller) error

// CloseFunc is a callback function for closing all listener clients.
type CloseFunc func(id string)

// Listener is an interface for network listeners. A network listener listens
// for incoming client connections and adds them to the server.
type Listener interface {
	SetConfig(*Config)           // set the listener config.
	Listen(s *system.Info) error // open the network address.
	Serve(EstablishFunc)         // starting actively listening for new connections.
	ID() string                  // return the id of the listener.
	Close(CloseFunc)             // stop and close the listener.
}

// Listeners contains the network listeners for the broker.
type Listeners struct {
	sync.RWMutex
	wg       sync.WaitGroup      // a waitgroup that waits for all listeners to finish.
	internal map[string]Listener // a map of active listeners.
	system   *system.Info        // pointers to system info.
}

// New returns a new instance of Listeners.
func New(s *system.Info) *Listeners {
	return &Listeners{
		internal: map[string]Listener{},
		system:   s,
	}
}

// Add adds a new listener to the listeners map, keyed on id.
func (l *Listeners) Add(val Listener) {
	l.Lock()
	l.internal[val.ID()] = val
	l.Unlock()
}

// Get returns the value of a listener if it exists.
func (l *Listeners) Get(id string) (Listener, bool) {
	l.RLock()
	val, ok := l.internal[id]
	l.RUnlock()
	return val, ok
}

// Len returns the length of the listeners map.
func (l *Listeners) Len() int {
	l.RLock()
	val := len(l.internal)
	l.RUnlock()
	return val
}

// Delete removes a listener from the internal map.
func (l *Listeners) Delete(id string) {
	l.Lock()
	delete(l.internal, id)
	l.Unlock()
}

// Serve starts a listener serving from the internal map.
func (l *Listeners) Serve(id string, establisher EstablishFunc) {
	l.RLock()
	listener := l.internal[id]
	l.RUnlock()

	go func(e EstablishFunc) {
		defer l.wg.Done()
		l.wg.Add(1)
		listener.Serve(e)
	}(establisher)
}

// ServeAll starts all listeners serving from the internal map.
func (l *Listeners) ServeAll(establisher EstablishFunc) {
	l.RLock()
	i := 0
	ids := make([]string, len(l.internal))
	for id := range l.internal {
		ids[i] = id
		i++
	}
	l.RUnlock()

	for _, id := range ids {
		l.Serve(id, establisher)
	}
}

// Close stops a listener from the internal map.
func (l *Listeners) Close(id string, closer CloseFunc) {
	l.RLock()
	listener := l.internal[id]
	l.RUnlock()
	listener.Close(closer)
}

// CloseAll iterates and closes all registered listeners.
func (l *Listeners) CloseAll(closer CloseFunc) {
	l.RLock()
	i := 0
	ids := make([]string, len(l.internal))
	for id := range l.internal {
		ids[i] = id
		i++
	}
	l.RUnlock()

	for _, id := range ids {
		l.Close(id, closer)
	}
	l.wg.Wait()
}
