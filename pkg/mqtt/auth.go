package mqtt

import (
	cmap "github.com/orcaman/concurrent-map"
)

// Auth is an example auth provider for the server.
type Auth struct {
	authFn func(user, password []byte) bool

	users cmap.ConcurrentMap
}

func (a *Auth) Authenticate(user, password []byte) bool {
	return a.authFn(user, password)
}

func (a *Auth) ACL(user []byte, topic string, write bool) bool {
	return true
}

func createAuth(authFn func(user, password []byte) bool) *Auth {
	return &Auth{
		authFn: authFn,
		users:  cmap.New(),
	}
}
