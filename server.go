package nrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/tehsphinx/nrpc/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const checkSubsInterval = 5 * time.Minute

// Server implements the grpc.ServiceRegistrar interface.
type Server struct {
	pub pubsub.Publisher
	sub pubsub.Subscriber
	log Logger

	subs     *subscriptions
	shutdown context.CancelFunc

	unaryInt    grpc.UnaryServerInterceptor
	streamInt   grpc.StreamServerInterceptor
	serviceInfo map[string]grpc.ServiceInfo
}

var _ grpc.ServiceRegistrar = (*Server)(nil)
var _ reflection.ServiceInfoProvider = (*Server)(nil)

// Run starts the server by subscribing to the registered endpoints.
func (s *Server) Run(ctx context.Context) error {
	if r := s.subs.subscribe(s.sub); r != nil {
		return r
	}
	if r := s.sub.Flush(); r != nil {
		return r
	}

	shutdownCtx, shutdown := context.WithCancel(ctx)
	s.shutdown = shutdown

	go func() {
		defer shutdown()

		if err := s.subs.watchSubscriptions(shutdownCtx); err != nil {
			s.log.Errorf("subscriptions watcher returned with error: %v", err)
		}
	}()
	return nil
}

// Listen starts the server by subscribing to the registered endpoints
// and blocks until closed or an error occurs.
func (s *Server) Listen(ctx context.Context) error {
	if r := s.subs.subscribe(s.sub); r != nil {
		return r
	}
	if r := s.sub.Flush(); r != nil {
		return r
	}

	shutdownCtx, shutdown := context.WithCancel(ctx)
	s.shutdown = shutdown
	defer shutdown()

	return s.subs.watchSubscriptions(shutdownCtx)
}

// Stop signals the Service to shutdown. Stopping is done when the Listen function returns.
func (s *Server) Stop() {
	s.shutdown()
}

// RegisterService implements the grpc.ServiceRegistrar interface.
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	prefix := "nrpc." + desc.ServiceName

	for _, mDesc := range desc.Methods {
		subject := prefix + "." + mDesc.MethodName

		s.subs.RegisterSubscription(subscription{
			endpoint: subject,
			queue:    desc.ServiceName,
			handler:  s.handleMethod(mDesc, impl),
		})
	}

	for _, sDesc := range desc.Streams {
		subject := prefix + "." + sDesc.StreamName

		s.subs.RegisterSubscription(subscription{
			endpoint: subject,
			queue:    desc.ServiceName,
			handler:  s.handleStream(sDesc, impl),
		})
	}

	s.registerServiceInfo(desc)
}

func (s *Server) registerServiceInfo(desc *grpc.ServiceDesc) {
	methods := make([]grpc.MethodInfo, 0, len(desc.Methods)+len(desc.Streams))
	for _, mDesc := range desc.Methods {
		methods = append(methods, grpc.MethodInfo{
			Name:           mDesc.MethodName,
			IsClientStream: false,
			IsServerStream: false,
		})
	}

	for _, sDesc := range desc.Streams {
		methods = append(methods, grpc.MethodInfo{
			Name:           sDesc.StreamName,
			IsClientStream: sDesc.ClientStreams,
			IsServerStream: sDesc.ServerStreams,
		})
	}

	s.serviceInfo[desc.ServiceName] = grpc.ServiceInfo{
		Methods:  methods,
		Metadata: desc.Metadata,
	}
}

// GetServiceInfo implements the grpc.ServiceInfoProvider interface.
func (s *Server) GetServiceInfo() map[string]grpc.ServiceInfo {
	return s.serviceInfo
}

func (s *Server) handleMethod(desc grpc.MethodDesc, impl interface{}) pubsub.Handler {
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

			return desc.Handler(impl, ctx, dec, s.unaryInt)
		}()
		if err != nil {
			s.respondErr(msg, err)
			return
		}

		payload, err := marshalUnaryRespMsg(msg.Subject(), resp.(proto.Message), transport.header, transport.trailer, true, false)
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

func (s *Server) handleStream(desc grpc.StreamDesc, impl interface{}) pubsub.Handler {
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

				if s.streamInt != nil {
					// pass the call through the stream interceptor
					info := &grpc.StreamServerInfo{
						FullMethod:     desc.StreamName,
						IsClientStream: desc.ClientStreams,
						IsServerStream: desc.ServerStreams,
					}
					return s.streamInt(impl, stream, info, desc.Handler)
				}

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

func contextTimeout(ctx context.Context, timeout int64) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Duration(timeout))
}
