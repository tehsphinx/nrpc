package nrpc

import (
	"google.golang.org/grpc/metadata"
)

func newServerTransport(method string) *serverTransport {
	return &serverTransport{
		method: method,
	}
}

type serverTransport struct {
	method  string
	header  metadata.MD
	trailer metadata.MD
}

// Method implements grpc.ServerTransportStream interface.
func (s serverTransport) Method() string {
	return s.method
}

// SetHeader implements grpc.ServerTransportStream interface.
func (s *serverTransport) SetHeader(md metadata.MD) error {
	s.header = md
	return nil
}

// SendHeader implements grpc.ServerTransportStream interface.
func (s serverTransport) SendHeader(md metadata.MD) error {
	return nil
}

// SetTrailer implements grpc.ServerTransportStream interface.
func (s *serverTransport) SetTrailer(md metadata.MD) error {
	s.trailer = md
	return nil
}
