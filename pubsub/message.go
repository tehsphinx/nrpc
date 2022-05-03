package pubsub

// Message defines a pubsub message.
type Message struct {
	Subject string
	Reply   string
	Data    []byte
}

// Reply defines a pubsub reply.
type Reply struct {
	Data []byte
}
