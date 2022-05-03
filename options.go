package nrpc

// Option defines an option for configuring the server.
type Option func(opt *options)

func getOptions(opts []Option) options {
	opt := options{
		logger: noopLogger{},
	}

	for _, o := range opts {
		o(&opt)
	}
	return opt
}

type options struct {
	logger Logger
}

// WithLogger sets the logger for the server.
func WithLogger(log Logger) Option {
	return func(opt *options) {
		opt.logger = log
	}
}
