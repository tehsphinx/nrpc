// Package testclient creates a new client for the testproto service used in tests.
package testclient

import (
	"github.com/tehsphinx/nrpc"
	"github.com/tehsphinx/nrpc/pubsub"
	"github.com/tehsphinx/nrpc/testproto"
)

// New creates a new test client.
func New(pub pubsub.Publisher, sub pubsub.Subscriber, opts ...nrpc.Option) testproto.TestClient {
	nrpcClient := nrpc.NewClient(pub, sub, opts...)
	return testproto.NewTestClient(nrpcClient)
}
