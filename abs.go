package wsrpc

// RPCTransport represents abstract transport for recv/send packages
type RPCTransport interface {
	Recv() (*Packet, error)
	Send(*Packet) error
	Close() error
}

// SessionProtocol represent abstract RPC protocol
type SessionProtocol interface {
	OnConnect(*RPCConn)
	OnDisconnect(error)
}
