package hanami

import (
	"strings"
	"sync"
)

// ClientSubscriptions is a map of subscriptions keyed on client id.
type ClientSubscriptions map[string]*Subscription

// Subscription contains the values for a topic subscription.
type Subscription struct {

	// Topic is the path of the subscription.
	Filter string

	// QOS is the QOS byte used in the subscription.
	QOS byte

	// Signed indicates that the subscription expects signed payloads.
	Signed bool

	// Callback is a subscription callback function called on message receipt.
	Callback Callback
}

// Subscriptions is a map of Subscription keyed on topic.
type Subscriptions struct {
	sync.RWMutex

	// Internal is a map of ClientSubscription keyed on topic filter.
	internal map[string]ClientSubscriptions
}

// NewSubscriptions returns a pointer to a new Subscriptions map.
func NewSubscriptions() Subscriptions {
	return Subscriptions{
		internal: map[string]ClientSubscriptions{},
	}
}

// get returns a map of sub-client subscriptions if they exists.
func (s *Subscriptions) get(filter string) (ClientSubscriptions, bool) {
	s.RLock()
	defer s.RUnlock()
	val, ok := s.internal[filter]
	return val, ok
}

// getByID returns the value of a sub-client subscription if it exists.
func (s *Subscriptions) getByID(filter, id string) (*Subscription, bool) {
	s.RLock()
	defer s.RUnlock()
	if _, ok := s.internal[filter]; !ok {
		return nil, ok
	}
	val, ok := s.internal[filter][id]
	return val, ok
}

// delete removes a filter and all sub-clients from the subscriptions.
func (s *Subscriptions) delete(filter string) {
	s.Lock()
	defer s.Unlock()
	delete(s.internal, filter)
}

// deleteByID removes a subclient id from a subscription filter.
func (s *Subscriptions) deleteByID(filter, id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.internal[filter], id)
}

// deleteAllByID removes all subscriptions matching a client id, returning a slice
// of effected filters.
func (s *Subscriptions) deleteAllByID(id string, isPrefix bool) []string {
	removed := make([]string, 0)
	s.Lock()
	defer s.Unlock()
	for filter, subc := range s.internal {
		for k := range subc {
			if k == id || isPrefix && strings.HasPrefix(k, id) {
				removed = append(removed, filter)
				delete(s.internal[filter], k)
			}
		}
	}
	return removed
}

// isEmpty returns true if a filter is empty.
func (s *Subscriptions) isEmpty(filter string) bool {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.internal[filter]; !ok {
		return true
	}
	return len(s.internal[filter]) == 0
}

// setByID stores the value of a subscription for a sub-client id.
func (s *Subscriptions) setByID(id string, val *Subscription) bool {
	var isNew bool
	s.Lock()
	defer s.Unlock()
	if _, ok := s.internal[val.Filter]; !ok {
		s.internal[val.Filter] = make(ClientSubscriptions)
		isNew = true
	}
	s.internal[val.Filter][id] = val
	return isNew
}

// MatchTopic checks if a given topic matches a filter, accounting for filter
// wildcards. Eg. filter /a/b/+/c == topic a/b/d/c.
func MatchTopic(filter string, topic string) (elements []string, matched bool) {
	filterParts := strings.Split(filter, "/")
	topicParts := strings.Split(topic, "/")
	elements = make([]string, 0)

	for i := 0; i < len(filterParts); i++ {
		if i >= len(topicParts) {
			matched = false
			return
		}

		if filterParts[i] == "+" {
			elements = append(elements, topicParts[i])
			continue
		}

		if filterParts[i] == "#" {
			matched = true
			elements = append(elements, strings.Join(topicParts[i:], "/"))
			return
		}

		if filterParts[i] != topicParts[i] {
			matched = false
			return
		}
	}

	return elements, true
}
