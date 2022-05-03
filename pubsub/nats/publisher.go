package nats

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/tehsphinx/nrpc/pubsub"
)

// Publisher returns a NATS wrapper implementing the pubsub.Publisher interface.
func Publisher(nats *nats.Conn) pubsub.Publisher {
	return &publisher{nats: nats}
}

type publisher struct {
	nats *nats.Conn
}

// Publish implements the pubsub.Publisher interface.
func (s *publisher) Publish(msg pubsub.Message) error {
	return s.nats.PublishMsg(&nats.Msg{
		Subject: msg.Subject,
		Reply:   msg.Reply,
		Data:    msg.Data,
	})
}

// Request implements the pubsub.Publisher interface.
func (s *publisher) Request(ctx context.Context, msg pubsub.Message) (pubsub.Message, error) {
	resp, err := s.nats.RequestMsgWithContext(ctx, &nats.Msg{
		Subject: msg.Subject,
		Data:    msg.Data,
	})
	if err != nil {
		return pubsub.Message{}, err
	}

	return pubsub.Message{
		Subject: resp.Subject,
		Data:    resp.Data,
	}, nil
}
