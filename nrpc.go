// Package nrpc implements gRPC transport over pubsub like NATS.
// The interfaces defined in `pubsub` package can be implemented
// for more pubsub solutions and then used with this module.
// This package implements the gRPC client and server interfaces
// of the official google.golang.org/grpc and can be used with grpc
// code generated with the official generator.
package nrpc

import (
	"time"

	"github.com/tehsphinx/nrpc/pubsub"
	"google.golang.org/grpc"
)

const (
	streamConnectTimeout = 5 * time.Second
	stuckTimeout         = 30 * time.Second
	randSubjectLen       = 10
)

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

		unaryInt:     opt.unaryInt,
		streamInt:    opt.streamInt,
		statsHandler: opt.statsHandler,
		serviceInfo:  map[string]grpc.ServiceInfo{},
	}
}
