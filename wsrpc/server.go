package wsrpc

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// RPCNotifier implements notifications sender from server to client
type RPCNotifier struct {
	protDetails *protocolDetails
	notifChan   chan *Packet
}

func (n *RPCNotifier) Notify(notification interface{}) error {
	// check type of notification
	nt := reflect.TypeOf(notification)
	if nt.Kind() == reflect.Ptr {
		nt = nt.Elem()
	}
	vt, ok := n.protDetails.notifications[nt.Name()]
	if !ok || vt != nt {
		return fmt.Errorf("Notification %s is not declared in protocol", reflect.TypeOf(notification))
	}

	// marshal notification to []byte
	buf, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	p := NewPacket(PT_NOTIFICATION, nt.Name(), buf)
	select {
	case n.notifChan <- p:
	default:
		go func() {
			n.notifChan <- p
		}()
	}

	return nil
}

//RPCServer implements RPC server protocol handler
type RPCServer struct {
	conns <-chan RPCTransport

	protocol    NewSessionFunc
	protDetails *protocolDetails

	finishCh chan struct{}

	log Logger
}

type NewSessionFunc func() SessionProtocol

func NewRPCServer(conns <-chan RPCTransport, f NewSessionFunc, log Logger) (*RPCServer, error) {
	rpc := &RPCServer{
		conns:    conns,
		finishCh: make(chan struct{}),
		log:      log,
	}

	p := f()
	pdetails, err := parseSessionProtocol(p)
	if err != nil {
		return nil, err
	}
	rpc.protDetails = pdetails
	rpc.protocol = f
	return rpc, nil

}

func (rpc *RPCServer) Run() {
	for conn := range rpc.conns {
		go rpc.procConn(conn)
	}
}

func (rpc *RPCServer) Close() {
	close(rpc.finishCh)
}

func (rpc *RPCServer) procConn(conn RPCTransport) {
	rpc.log.Debugf("new connection established")
	prot := rpc.protocol()
	respCh := make(chan *Packet)
	notifier := &RPCNotifier{rpc.protDetails, respCh}
	prot.OnConnect(conn, notifier)

	readFunc := func() {
		for {
			packet, err := conn.Recv()
			if err != nil {
				rpc.log.Debugf("returning rpc.readFunc() with err: %s", err.Error())
				prot.OnDisconnect(err)
				return
			}

			go rpc.callMethod(prot, packet, respCh) //TODO maybe worker pool should be implemented
		}
	}
	go readFunc()

	for {
		select {
		case retPacket := <-respCh:
			err := conn.Send(retPacket)
			if err != nil {
				// logging error
				rpc.log.Errorf("can't send packet to client: %s", err.Error())
				return
			}

		case <-rpc.finishCh:
			conn.Close()
			return
		}
	}
}

func (rpc *RPCServer) callMethod(p SessionProtocol, packet *Packet, respCh chan<- *Packet) {
	m, ok := rpc.protDetails.methods[packet.Header.Method] //FIXME lock (?)
	if !ok {
		respCh <- packet.Error(
			fmt.Errorf("no method %s found", packet.Header.Method),
		)
		return
	}

	inV := reflect.New(m.inType)
	err := json.Unmarshal(packet.Body, inV.Interface())
	if err != nil {
		respCh <- packet.Error(err)
		return
	}

	ret := m.funcVal.Call([]reflect.Value{reflect.ValueOf(p), inV})

	var buf []byte
	if !ret[1].IsNil() { // check error
		respCh <- packet.Error(ret[1].Interface().(error))
		return
	}

	buf, err = json.Marshal(ret[0].Interface())
	if err != nil {
		respCh <- packet.Error(err)
		return
	}

	h := Header{
		MessageId: packet.Header.MessageId,
		Type:      PT_RESPONSE,
		Method:    packet.Header.Method,
	}
	respCh <- &Packet{Header: h, Body: buf}
}
