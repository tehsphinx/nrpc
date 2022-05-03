package nrpc

import (
	"sync"

	"github.com/tehsphinx/nrpc/pubsub"
)

func newSubscriptions(log Logger) *subscriptions {
	return &subscriptions{
		log:  log,
		subs: make(map[string]pubsub.Subscription),
	}
}

type subscriptions struct {
	log Logger

	m    sync.RWMutex
	subs map[string]pubsub.Subscription
}

func (s *subscriptions) register(name string, sub pubsub.Subscription) {
	s.m.Lock()
	defer s.m.Unlock()

	if subscr, ok := s.subs[name]; ok {
		_ = subscr.Unsubscribe()
		s.log.Infof("un-subscribed: subject => %v: subscription with same name", name)
	}
	s.subs[name] = sub
}
