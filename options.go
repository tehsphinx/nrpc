package nrpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// Option defines an option for configuring the server.
type Option func(opt *options)

func getOptions(opts []Option) options {
	opt := options{
		logger:       noopLogger{},
		statsHandler: noopStatsHandler{},
	}

	for _, o := range opts {
		o(&opt)
	}
	return opt
}

type options struct {
	logger Logger

	unaryInt     grpc.UnaryServerInterceptor
	streamInt    grpc.StreamServerInterceptor
	statsHandler stats.Handler
}

// WithLogger sets the logger for the client or server.
func WithLogger(log Logger) Option {
	return func(opt *options) {
		opt.logger = log
	}
}

// UnaryInterceptor returns a ServerOption that sets the UnaryServerInterceptor for the
// server. Only one unary interceptor can be installed. The construction of multiple
// interceptors (e.g., chaining) can be implemented at the caller.
func UnaryInterceptor(i grpc.UnaryServerInterceptor) Option {
	return func(opt *options) {
		if opt.unaryInt != nil {
			panic("nrpc: The unary server interceptor was already set and may not be reset.")
		}
		opt.unaryInt = i
	}
}

// StreamInterceptor returns a ServerOption that sets the StreamServerInterceptor for the
// server. Only one stream interceptor can be installed.
func StreamInterceptor(i grpc.StreamServerInterceptor) Option {
	return func(opt *options) {
		if opt.streamInt != nil {
			panic("nrpc: The stream server interceptor was already set and may not be reset.")
		}
		opt.streamInt = i
	}
}

// StatsHandler returns a ServerOption that sets the StatsHandler for the server.
// It can be used to add tracing, metrics, etc.
func StatsHandler(handler stats.Handler) Option {
	return func(opt *options) {
		opt.statsHandler = handler
	}
}
