package testproto

import (
	"fmt"
	"net"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// NewTestConn creates a nats test connection and returns a shutdown function to be deferred.
func NewTestConn() (conn *nats.Conn, shutdown func(), err error) {
	port, err := getFreePort(3)
	if err != nil {
		return nil, nil, fmt.Errorf("no free port found")
	}

	opts := server.Options{
		Host: "localhost",
		Port: port,
		// NoLog:  true,
		// NoSigs: true,
	}
	gnatsd, err := server.NewServer(&opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create nats server: %w", err)
	}
	gnatsd.Start()

	conn, err = nats.Connect(gnatsd.Addr().String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to nats server: %w", err)
	}
	if r := conn.Flush(); r != nil {
		return nil, nil, fmt.Errorf("failed to reach nats server: %w", r)
	}

	return conn, func() {
		conn.Close()
		gnatsd.Shutdown()
	}, nil
}

func getFreePort(n int) (port int, err error) {
	for i := 0; i < n; i++ {
		if port, err = getPort(); err == nil {
			return port, err
		}
	}
	return 0, err
}

func getPort() (port int, err error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	port = ln.Addr().(*net.TCPAddr).Port
	err = ln.Close()
	return port, err
}
