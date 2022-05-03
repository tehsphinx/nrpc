package pubsub

import (
	"context"
)

type Subscriber interface {
	Subscribe(subject, queue string, handler Handler) (Subscription, error)
	SubscribeAsync(subject, queue string, handler Handler) (Subscription, error)
	Flush() error
}

type Handler func(ctx context.Context, msg Replier)

type Replier interface {
	Subject() string
	Data() []byte

	Reply(msg Reply) error
}

type Subscription interface {
	Unsubscribe() error
}
