package nats

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/tehsphinx/nrpc/pubsub"
)

// Subscriber returns a NATS wrapper implementing the pubsub.Subscriber interface.
func Subscriber(nats *nats.Conn) pubsub.Subscriber {
	return &subscriber{nats: nats}
}

type subscriber struct {
	nats *nats.Conn
}

// Subscribe implements the pubsub.Subscriber interface.
func (s *subscriber) Subscribe(subject, queue string, handler pubsub.Handler) (pubsub.Subscription, error) {
	return s.nats.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		handler(context.Background(), message{msg: msg})
	})
}

// SubscribeAsync implements the pubsub.Subscriber interface.
func (s *subscriber) SubscribeAsync(subject, queue string, handler pubsub.Handler) (pubsub.Subscription, error) {
	return s.nats.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		go func(msg *nats.Msg) {
			handler(context.Background(), message{msg: msg})
		}(msg)
	})
}

// Flush implements the pubsub.Subscriber interface.
func (s *subscriber) Flush() error {
	return s.nats.Flush()
}
