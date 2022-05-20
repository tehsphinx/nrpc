package nrpc

import (
	"context"
	"errors"
	"time"

	"github.com/tehsphinx/nrpc/pubsub"
)

type subscription struct {
	endpoint string
	queue    string
	handler  pubsub.Handler
}

func newSubscriptions(log Logger) *subscriptions {
	return &subscriptions{
		log:  log,
		subs: make(map[string]pubsub.Subscription),
	}
}

type subscriptions struct {
	log Logger

	defs []subscription
	subs map[string]pubsub.Subscription
}

// RegisterSubscription registers a subscription.
func (s *subscriptions) RegisterSubscription(sub subscription) {
	s.defs = append(s.defs, sub)
}

func (s *subscriptions) subscribe(subscriber pubsub.Subscriber) error {
	for _, def := range s.defs {
		sub, err := subscriber.SubscribeAsync(def.endpoint, def.queue, def.handler)
		if err != nil {
			return err
		}

		if subscr, ok := s.subs[def.endpoint]; ok {
			_ = subscr.Unsubscribe()
			s.log.Infof("un-subscribed: subject => %v: subscription with same name", def.endpoint)
		}
		s.subs[def.endpoint] = sub

		s.log.Infof("Subscribed: subject => %v, queue => %v", def.endpoint, def.queue)
	}
	return nil
}

func (s *subscriptions) watchSubscriptions(ctx context.Context) error {
	defer s.closeSubscriptions()

	tick := time.NewTicker(checkSubsInterval)
	for {
		select {
		case <-tick.C:
			for _, sub := range s.subs {
				if sub.IsValid() {
					continue
				}
				return errors.New("subscription was closed unexpectedly")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *subscriptions) closeSubscriptions() {
	for _, sub := range s.subs {
		if r := sub.Unsubscribe(); r != nil {
			s.log.Infof("error closing subscription: %v", r)
		}
	}
}
