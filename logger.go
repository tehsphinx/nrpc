package nrpc

import "log"

// Logger defines the interface for logging.
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	// Debug(args ...interface{})
	// Debugf(format string, args ...interface{})
	// Warn(args ...interface{})
	// Warnf(format string, args ...interface{})
	// Fatal(args ...interface{})
	// Fatalf(format string, args ...interface{})
}

var _ Logger = (*noopLogger)(nil)
var _ Logger = (*StandardLogger)(nil)

type noopLogger struct{}

// Info implements the Logger interface
func (n noopLogger) Info(...interface{}) {}

// Infof implements the Logger interface.
func (n noopLogger) Infof(string, ...interface{}) {}

// Error implements the Logger interface.
func (n noopLogger) Error(...interface{}) {}

// Errorf implements the Logger interface.
func (n noopLogger) Errorf(string, ...interface{}) {}

// StandardLogger implements the Logger interface using the standard library logger.
type StandardLogger struct{}

// Info implements the Logger interface
func (d StandardLogger) Info(args ...interface{}) {
	log.Println(args...)
}

// Infof implements the Logger interface.
func (d StandardLogger) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Error implements the Logger interface.
func (d StandardLogger) Error(args ...interface{}) {
	d.Info(args...)
}

// Errorf implements the Logger interface.
func (d StandardLogger) Errorf(format string, args ...interface{}) {
	d.Infof(format, args...)
}
