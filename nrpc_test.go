package nrpc_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/tehsphinx/nrpc"
	"github.com/tehsphinx/nrpc/pubsub/nats"
	"github.com/tehsphinx/nrpc/testproto"
	"github.com/tehsphinx/nrpc/testproto/testclient"
	"github.com/tehsphinx/nrpc/testproto/testserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnary(t *testing.T) {
	asrt := is.New(t)
	ctx := context.Background()
	logger := nrpc.StandardLogger{}

	conn, shutdown, err := testproto.NewTestConn()
	asrt.NoErr(err)
	defer shutdown()

	pub := nats.Publisher(conn)
	sub := nats.Subscriber(conn)

	server, err := testserver.New(pub, sub, nrpc.WithLogger(logger))
	asrt.NoErr(err)
	client := testclient.New(pub, sub, nrpc.WithLogger(logger))
	_ = server

	t.Run("call", func(t *testing.T) {
		asrt := asrt.New(t)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// outbound header
		md := metadata.New(map[string]string{"heady": "head1"})
		ctx = metadata.NewOutgoingContext(ctx, md)

		// inbound header
		var header, trailer metadata.MD
		resp, err := client.Unary(ctx, &testproto.UnaryReq{
			Msg: "Hello via NRPC",
		}, grpc.Header(&header), grpc.Trailer(&trailer))

		asrt.NoErr(err)
		asrt.Equal(resp.Msg, "Hello back!")
		asrt.Equal(header.Get("heady"), []string{"head1"})
		asrt.Equal(header.Get("srv-key"), []string{"srv-value"})

		// TODO: test trailer
		// asrt.Equal(trailer.Get("traily"), []string{"t-value"})
	})
}

func TestServerStream(t *testing.T) {
	asrt := is.New(t)
	ctx := context.Background()
	logger := nrpc.StandardLogger{}

	conn, shutdown, err := testproto.NewTestConn()
	asrt.NoErr(err)
	defer shutdown()

	pub := nats.Publisher(conn)
	sub := nats.Subscriber(conn)

	server, err := testserver.New(pub, sub, nrpc.WithLogger(logger))
	asrt.NoErr(err)
	client := testclient.New(pub, sub, nrpc.WithLogger(logger))
	_ = server

	t.Run("call", func(t *testing.T) {
		asrt := asrt.New(t)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// outbound header
		md := metadata.New(map[string]string{"heady": "head1"})
		ctx = metadata.NewOutgoingContext(ctx, md)

		stream, err := client.ServerStream(ctx, &testproto.ServerStreamReq{
			Msg: "Hello via NRPC",
		})
		asrt.NoErr(err)

		var i int
		for {
			msg, err := stream.Recv()
			if err != nil && err == io.EOF {
				break
			}
			asrt.NoErr(err)

			i++
			asrt.Equal(msg.Msg, fmt.Sprintf("Hello back! %d", i))
		}

		md, err = stream.Header()
		asrt.NoErr(err)
		asrt.Equal(md.Get("heady"), []string{"head1"})
		asrt.Equal(md.Get("srv-key"), []string{"srv-value"})

		// TODO: test trailer
		// md = stream.Trailer()
		// asrt.Equal(md.Get("traily"), []string{"t-value"})
	})
}

func TestClientStream(t *testing.T) {
	asrt := is.New(t)
	ctx := context.Background()
	logger := nrpc.StandardLogger{}

	conn, shutdown, err := testproto.NewTestConn()
	asrt.NoErr(err)
	defer shutdown()

	pub := nats.Publisher(conn)
	sub := nats.Subscriber(conn)

	server, err := testserver.New(pub, sub, nrpc.WithLogger(logger))
	asrt.NoErr(err)
	client := testclient.New(pub, sub, nrpc.WithLogger(logger))
	_ = server

	t.Run("call", func(t *testing.T) {
		asrt := asrt.New(t)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// outbound header
		md := metadata.New(map[string]string{"heady": "head1"})
		ctx = metadata.NewOutgoingContext(ctx, md)

		stream, err := client.ClientStream(ctx)
		asrt.NoErr(err)

		for i := 0; i < 5; i++ {
			err := stream.Send(&testproto.ClientStreamReq{
				Msg: fmt.Sprintf("Hello via NRPC %d", i+1),
			})
			asrt.NoErr(err)
		}

		resp, err := stream.CloseAndRecv()
		asrt.NoErr(err)
		asrt.Equal(resp.Msg, "Hello back!")

		md, err = stream.Header()
		asrt.NoErr(err)
		asrt.Equal(md.Get("heady"), []string{"head1"})
		asrt.Equal(md.Get("srv-key"), []string{"srv-value"})

		// TODO: test trailer
		// md = stream.Trailer()
		// asrt.Equal(md.Get("traily"), []string{"t-value"})
	})
}