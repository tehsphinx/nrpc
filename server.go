package nrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/tehsphinx/nrpc/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	pub pubsub.Publisher
	sub pubsub.Subscriber
	log Logger

	subs   *subscriptions
	regErr error
}

func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	prefix := "nrpc." + desc.ServiceName

	for _, mDesc := range desc.Methods {
		subject := prefix + "." + mDesc.MethodName

		sub, err := s.sub.SubscribeAsync(subject, desc.ServiceName, s.handleMethod(mDesc, impl))
		if err != nil {
			s.regErr = err
			return
		}

		s.subs.register(subject, sub)
		s.log.Infof("Subscribed: subject => %v, queue => %v", subject, desc.ServiceName)
	}

	for _, sDesc := range desc.Streams {
		subject := prefix + "." + sDesc.StreamName

		sub, err := s.sub.SubscribeAsync(subject, desc.ServiceName, s.handleStream(sDesc, impl))
		if err != nil {
			s.regErr = err
			return
		}

		s.subs.register(subject, sub)
		s.log.Infof("Subscribed: subject => %v, queue => %v", subject, desc.ServiceName)
	}

	if r := s.sub.Flush(); r != nil {
		s.regErr = r
		return
	}
}

func (s *Server) handleMethod(desc grpc.MethodDesc, impl interface{}) func(context.Context, pubsub.Replier) {
	return func(ctx context.Context, msg pubsub.Replier) {
		req, err := unmarshalReq(msg.Data())
		if err != nil {
			s.respondErr(msg, err)
			return
		}

		transport := newServerTransport(desc.MethodName)
		ctx = grpc.NewContextWithServerTransportStream(ctx, transport)
		ctx = metadata.NewIncomingContext(ctx, toMD(req.Header))
		ctx, cancel := contextTimeout(ctx, req.Timeout)
		defer cancel()

		dec := func(target interface{}) error {
			// nolint: forcetypeassert
			return proto.Unmarshal(req.Data, target.(proto.Message))
		}

		resp, err := func() (_ interface{}, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("server.handleMethod: panic recovered: %v", r)
				}
			}()

			return desc.Handler(impl, ctx, dec, nil)
		}()
		if err != nil {
			s.respondErr(msg, err)
			return
		}

		payload, err := marshalRespMsg(resp.(proto.Message), transport.header, transport.trailer, true, false)
		if err != nil {
			s.respondErr(msg, err)
			return
		}

		s.reply(msg, payload)
	}
}

func (s *Server) respondErr(msg pubsub.Replier, resErr error) {
	errStatus, ok := status.FromError(resErr)
	if !ok {
		errStatus = status.FromContextError(resErr)
		// updates the resErr to the status error. Needed if below TODO is implemented
		// resErr = errStatus.Err()
	}

	// TODO: inject external error handler for logging, tracing, etc.

	payload, err := marshalProto(msg.Subject(), errStatus.Proto(), MessageType_Error)
	if err != nil {
		s.log.Errorf("Failed to marshal error response: %v", err)
		return
	}

	s.reply(msg, payload)
}

func (s *Server) reply(msg pubsub.Replier, payload []byte) {
	s.log.Infof("Reply: subject => %v", msg.Subject())
	if r := msg.Reply(pubsub.Reply{
		Data: payload,
	}); r != nil {
		s.log.Errorf("Failed to reply: %v", r)
		return
	}
}

func (s *Server) handleStream(desc grpc.StreamDesc, impl interface{}) func(context.Context, pubsub.Replier) {
	return func(ctx context.Context, msg pubsub.Replier) {
		// dbg.Green("server.handleStream", msg.Subject(), msg.Data())

		stream := newServerStream(s.pub, s.sub, s.log, desc)
		if r := stream.Subscribe(ctx, msg.Data()); r != nil {
			s.respondErr(msg, fmt.Errorf("failed to subscribe: %w", r))
			return
		}
		go func() {
			if r := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("server.handleStream: panic recovered: %v", r)
					}
				}()

				return desc.Handler(impl, stream)
			}(); r != nil {
				stream.CloseWithError(r)
				return
			}

			stream.Close()
		}()

		if r := msg.Reply(pubsub.Reply{
			Data: nil,
		}); r != nil {
			s.respondErr(msg, fmt.Errorf("failed to reply: %w", r))
			return
		}
	}
}

// RegistrationErr returns an error that occurred during registration.
func (s Server) RegistrationErr() error {
	return s.regErr
}

func contextTimeout(ctx context.Context, timeout int64) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Duration(timeout))
}
