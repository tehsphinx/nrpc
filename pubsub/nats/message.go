package nats

import (
	"github.com/nats-io/nats.go"
	"github.com/tehsphinx/nrpc/pubsub"
)

type message struct {
	msg *nats.Msg
}

var _ pubsub.Replier = (*message)(nil)

// Subject implements the Msg interface.
func (s message) Subject() string {
	return s.msg.Subject
}

// Data implements the Msg interface.
func (s message) Data() []byte {
	return s.msg.Data
}

// Reply implemets the Msg interface.
func (s message) Reply(msg pubsub.Reply) error {
	return s.msg.Respond(msg.Data)
}
