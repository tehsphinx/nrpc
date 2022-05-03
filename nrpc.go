package nrpc

import "github.com/tehsphinx/nrpc/pubsub"

// NewClient creates a new pub-sub based grpc client.
func NewClient(pub pubsub.Publisher, sub pubsub.Subscriber, opts ...Option) *Client {
	opt := getOptions(opts)

	return &Client{
		pub: pub,
		sub: sub,
		log: opt.logger,
	}
}

// NewServer creates a new pub-sub based grpc server.
func NewServer(pub pubsub.Publisher, sub pubsub.Subscriber, opts ...Option) *Server {
	opt := getOptions(opts)

	return &Server{
		pub:  pub,
		sub:  sub,
		log:  opt.logger,
		subs: newSubscriptions(opt.logger),
	}
}
