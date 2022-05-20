package nrpc

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/tehsphinx/nrpc/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func newServerStream(pub pubsub.Publisher, sub pubsub.Subscriber, log Logger, desc grpc.StreamDesc) *serverStream {
	return &serverStream{
		pub:    pub,
		sub:    sub,
		log:    log,
		desc:   desc,
		chRecv: make(chan *recvMsg, 1),
	}
}

type serverStream struct {
	pub  pubsub.Publisher
	sub  pubsub.Subscriber
	log  Logger
	desc grpc.StreamDesc

	ctx         context.Context
	cancel      context.CancelFunc
	respSubj    string
	chRecv      chan *recvMsg
	sendHeader  metadata.MD
	sendTrailer metadata.MD
}

// SetHeader sets the header metadata. It may be called multiple times.
// When call multiple times, all the provided metadata will be merged.
// All the metadata will be sent out when one of the following happens:
//  - ServerStream.SendHeader() is called;
//  - The first response is sent out;
//  - An RPC status is sent out (error or success).
func (s *serverStream) SetHeader(md metadata.MD) error {
	if s.sendHeader == nil {
		s.sendHeader = md
		return nil
	}
	for k, vals := range md {
		s.sendHeader[k] = append(s.sendHeader[k], vals...)
	}
	return nil
}

// SendHeader sends the header metadata.
// The provided md and headers set by SetHeader() will be sent.
// It fails if called multiple times.
func (s *serverStream) SendHeader(md metadata.MD) error {
	if r := s.SetHeader(md); r != nil {
		return r
	}
	return s.sendMsg(nil, false, true)
}

// SetTrailer sets the trailer metadata which will be sent with the RPC status.
// When called more than once, all the provided metadata will be merged.
func (s *serverStream) SetTrailer(md metadata.MD) {
	if s.sendTrailer == nil {
		s.sendTrailer = md
		return
	}
	for k, vals := range md {
		s.sendTrailer[k] = append(s.sendTrailer[k], vals...)
	}
}

// Context returns the context for this stream.
func (s *serverStream) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

// SendMsg sends a message. On error, SendMsg aborts the stream and the
// error is returned directly.
//
// SendMsg blocks until:
//   - There is sufficient flow control to schedule m with the transport, or
//   - The stream is done, or
//   - The stream breaks.
//
// SendMsg does not wait until the message is received by the client. An
// untimely stream closure may result in lost messages.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not safe
// to call SendMsg on the same stream in different goroutines.
func (s *serverStream) SendMsg(m interface{}) error {
	// nolint: forcetypeassert
	args := m.(proto.Message)
	return s.sendMsg(args, false, false)
}

// Close closes the stream with OK status.
func (s *serverStream) Close() {
	if r := s.sendMsg(nil, true, false); r != nil {
		s.log.Errorf("failed to close stream: %w", r)
		return
	}
}

// CloseWithError closes the stream with the specified error.
func (s *serverStream) CloseWithError(err error) {
	stats := status.Convert(err)
	if r := s.sendMsg(stats.Proto(), true, false); r != nil {
		s.log.Errorf("failed to close stream with error: %w", r)
		return
	}
}

func (s *serverStream) sendMsg(args proto.Message, eos, headerOnly bool) (err error) {
	defer func() {
		if err != nil || eos {
			s.cancel()
		}
	}()
	payload, err := marshalRespMsg(args, s.sendHeader, s.sendTrailer, eos, headerOnly)
	if err != nil {
		return err
	}

	msg := pubsub.Message{
		Subject: s.respSubj,
		Data:    payload,
	}
	s.sendHeader = nil

	return s.pub.Publish(msg)
}

// RecvMsg blocks until it receives a message into m or the stream is
// done. It returns io.EOF when the client has performed a CloseSend. On
// any non-EOF error, the stream is aborted and the error contains the
// RPC status.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not
// safe to call RecvMsg on the same stream in different goroutines.
func (s *serverStream) RecvMsg(target interface{}) (err error) {
	defer func() {
		if err != nil {
			s.cancel()
		}
	}()
	req, err := s.recvMsg(target)
	if err != nil {
		return err
	}
	// if req.Header != nil {
	// 	s.ctx = metadata.NewIncomingContext(s.ctx, toMD(req.Header))
	// }
	if req.Eos {
		return io.EOF
	}

	return nil
}

func (s *serverStream) recvMsg(target interface{}) (*Request, error) {
	var recv *recvMsg
	select {
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case recv = <-s.chRecv:
	}

	req, err := recv.request()
	if err != nil {
		return nil, err
	}

	// nolint: forcetypeassert
	if r := proto.Unmarshal(req.Data, target.(proto.Message)); r != nil {
		return nil, r
	}
	return req, nil
}

// Subscribe subscribes to the client stream.
func (s *serverStream) Subscribe(ctx context.Context, reqData []byte) error {
	queue := "receive"

	req, err := unmarshalReq(reqData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal request message: %w", err)
	}
	s.respSubj = req.RespSubject
	ctx = metadata.NewIncomingContext(ctx, toMD(req.Header))
	s.ctx, s.cancel = context.WithCancel(ctx)

	s.log.Infof("Subscribed Stream (server): Subject => %s, Queue => %s", req.ReqSubject, queue)
	sub, err := s.sub.Subscribe(req.ReqSubject, queue, func(ctx context.Context, msg pubsub.Replier) {
		// dbg.Cyan("client -> server (received)", msg.Subject(), msg.Data())
		select {
		case <-s.ctx.Done():
		case s.chRecv <- &recvMsg{ctx: ctx, data: msg.Data()}:
		default:
			select {
			case <-s.ctx.Done():
			case <-ctx.Done():
				s.cancel()
			case s.chRecv <- &recvMsg{ctx: ctx, data: msg.Data()}:
			case <-time.After(stuckTimeout):
				s.log.Errorf("Stream: Subject => %s, Queue => %s: closing stream: "+
					"server stream consumer stuck for 30sec", s.respSubj, queue)
				s.cancel()
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	go func() {
		<-s.ctx.Done()
		_ = sub.Unsubscribe()
	}()

	s.chRecv <- &recvMsg{
		ctx: ctx,
		req: req,
	}
	return nil
}
