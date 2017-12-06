package wsrpc

import "io"

// RPCTransport represents abstract transport for recv/send packages
type RPCTransport interface {
	Recv() <-chan *Packet
	Send(*Packet) error
	Close() error
	Closed() <-chan error
}

// SessionProtocol represent abstract RPC protocol
type SessionProtocol interface {
	OnConnect(io.Closer, *RPCNotifier)
	OnDisconnect(error)
}
