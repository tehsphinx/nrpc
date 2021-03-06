// Package testserver implements the testproto service used in tests.
package testserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/tehsphinx/nrpc"
	"github.com/tehsphinx/nrpc/pubsub"
	"github.com/tehsphinx/nrpc/testproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// New creates a new test server.
func New(pub pubsub.Publisher, sub pubsub.Subscriber, opts ...nrpc.Option) (*nrpc.Server, *Server, error) {
	impl := &Server{msgCount: 5}

	rpcServer := nrpc.NewServer(pub, sub, opts...)
	testproto.RegisterTestServer(rpcServer, impl)
	if r := rpcServer.Run(context.Background()); r != nil {
		return nil, nil, r
	}

	return rpcServer, impl, nil
}

// Server is the server implementation of the testproto.TestServiceServer interface.
type Server struct {
	testproto.UnimplementedTestServer
	msgCount int
}

var _ testproto.TestServer = (*Server)(nil)

// SetMsgCount sets the number of messages to send in the tests.
func (s *Server) SetMsgCount(count int) {
	s.msgCount = count
}

// Unary implements a unary RPC method for testing.
func (s Server) Unary(ctx context.Context, req *testproto.UnaryReq) (*testproto.UnaryResp, error) {
	if req.Msg != "Hello via NRPC" {
		return nil, status.Error(codes.InvalidArgument, "invalid message")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("failed to get metadata from context")
		return nil, status.Error(codes.DataLoss, "missing metadata")
	}

	md.Set("srv-key", "srv-value")
	if r := grpc.SetHeader(ctx, md); r != nil {
		log.Println("unable to send header back to client")
		return nil, status.Errorf(codes.Internal, "unable to send header: %v", r)
	}
	if r := grpc.SetTrailer(ctx, metadata.Pairs("traily", "t-value")); r != nil {
		return nil, status.Errorf(codes.Internal, "unable to send trailer: %v", r)
	}

	return &testproto.UnaryResp{
		Msg: "Hello back!",
	}, nil
}

// ServerStream implements a server streaming RPC method for testing.
func (s Server) ServerStream(req *testproto.ServerStreamReq, stream testproto.Test_ServerStreamServer) error {
	if req.Msg != "Hello via NRPC" {
		return status.Error(codes.InvalidArgument, "invalid message")
	}
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		log.Println("failed to get metadata from context")
		return status.Error(codes.DataLoss, "missing metadata")
	}

	md.Set("srv-key", "srv-value")
	if r := stream.SetHeader(md); r != nil {
		return r
	}
	stream.SetTrailer(metadata.Pairs("traily", "t-value"))

	for i := 0; i < s.msgCount; i++ {
		if r := stream.Send(&testproto.ServerStreamResp{
			Msg: fmt.Sprintf("Hello back! %d", i+1),
		}); r != nil {
			return r
		}
	}

	return nil
}

// ClientStream implements a client streaming RPC method for testing.
func (s Server) ClientStream(stream testproto.Test_ClientStreamServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		log.Println("failed to get metadata from context")
		return status.Error(codes.DataLoss, "missing metadata")
	}

	stream.SetTrailer(metadata.Pairs("traily", "t-value"))

	var i int
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		i++
		if req.Msg != fmt.Sprintf("Hello via NRPC %d", i) {
			return status.Error(codes.InvalidArgument, "invalid message")
		}
	}

	md.Set("srv-key", "srv-value")
	if r := stream.SetHeader(md); r != nil {
		return r
	}
	return stream.SendAndClose(&testproto.ClientStreamResp{
		Msg: "Hello back!",
	})
}

// BiDiStream implements a bidirectional streaming RPC method for testing.
func (s Server) BiDiStream(stream testproto.Test_BiDiStreamServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		log.Println("failed to get metadata from context")
		return status.Error(codes.DataLoss, "missing metadata")
	}

	md.Set("srv-key", "srv-value")
	if r := stream.SendHeader(md); r != nil {
		return r
	}

	stream.SetTrailer(metadata.Pairs("traily", "t-value"))

	var i int
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		i++
		if req.Msg != fmt.Sprintf("Hello via NRPC %d", i) {
			return status.Error(codes.InvalidArgument, "invalid message")
		}
		if r := stream.Send(&testproto.BiDiStreamResp{
			Msg: fmt.Sprintf("Hello back! %d", i),
		}); r != nil {
			return r
		}
	}

	return nil
}
