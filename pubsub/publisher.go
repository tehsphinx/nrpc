package pubsub

import "context"

type Publisher interface {
	Publish(msg Message) error
	Request(ctx context.Context, msg Message) (Message, error)
}
