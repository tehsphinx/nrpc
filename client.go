package nrpc

import (
	"context"
	"strings"
	"time"

	"github.com/tehsphinx/nrpc/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Client implements a pub-sub based grpc client.
type Client struct {
	pub pubsub.Publisher
	sub pubsub.Subscriber
	log Logger
}

// Invoke performs a unary RPC and returns after the response is received
// into reply.
func (s *Client) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	timeout := timeoutFromCtx(ctx)
	if timeout < 0 {
		return ctx.Err()
	}

	payload, err := marshalReqMsg(ctx, args.(proto.Message), "", "", timeout)
	if err != nil {
		return err
	}

	req := pubsub.Message{
		Subject: methodSubj(method),
		Data:    payload,
	}

	s.log.Infof("Request: subject => %v", req.Subject)
	res, err := s.pub.Request(ctx, req)
	if err != nil {
		return err
	}

	resp, err := unmarshalRespMsg(res.Data, reply.(proto.Message))
	if err != nil {
		return err
	}
	applyRespToOptions(opts, resp)
	return nil
}

func timeoutFromCtx(ctx context.Context) int64 {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline).Nanoseconds()
	}
	return 0
}

// NewStream begins a streaming RPC.
func (s *Client) NewStream(ctx context.Context, _ *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	stream := newClientStream(ctx, s.pub, s.sub, s.log, method, opts)
	if r := stream.Subscribe(); r != nil {
		return nil, r
	}
	return stream, nil
}

func applyRespToOptions(opts []grpc.CallOption, resp *Response) {
	for _, opt := range opts {
		switch o := opt.(type) {
		case grpc.HeaderCallOption:
			md := toMD(resp.Header)
			*o.HeaderAddr = md
		case grpc.TrailerCallOption:
			md := toMD(resp.Trailer)
			*o.TrailerAddr = md
		}
	}
}

func methodSubj(method string) string {
	return "nrpc" + strings.ReplaceAll(method, "/", ".")
}
