package main

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

type RPCTransport interface {
	Recv() <-chan *Packet
	Send(*Packet) error
	Close() error
	Closed() <-chan error
}

// ---------------------------------------------------------------------------

type RPCNotifier struct {
	notifChan chan *Packet
}

func (n *RPCNotifier) Notify(notification interface{}) error {
	buf, err := json.Marshal(notification) //FIXME check Type
	if err != nil {
		return err
	}
	go func() {
		n.notifChan <- NewPacket(PT_NOTIFICATION, "FIXME", buf)
	}()
	return nil
}

func (n *RPCNotifier) wait() <-chan *Packet {
	return n.notifChan
}

// ---------------------------------------------------------------------------

type SessionProtocol interface {
	OnConnect(io.Closer, *RPCNotifier)
	OnDisconnect(error)
}

// ---------------------------------------------------------------------------

type methodDetails struct {
	funcVal reflect.Value
	inType  reflect.Type
	outType reflect.Type
}

type RPCServer struct {
	conns <-chan RPCTransport

	protocol NewSessionFunc
	methods  map[string]methodDetails
}

type NewSessionFunc func() SessionProtocol

func NewRPCServer(conns <-chan RPCTransport, f NewSessionFunc) (*RPCServer, error) {
	rpc := &RPCServer{
		conns:   conns,
		methods: make(map[string]methodDetails),
	}

	errorInterface := reflect.TypeOf((*error)(nil)).Elem()

	p := f()
	pType := reflect.TypeOf(p)
	if pType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("pointer to protocol instance expected")
	}
	for i := 0; i < pType.NumMethod(); i++ {
		m := pType.Method(i)
		switch m.Name {
		case "OnConnect", "OnDisconnect":
			continue
		default:
		}
		//fmt.Printf("method #%d: name=%s, type=%s, func=%s\n", i, m.Name, m.Type, m.Func)

		// check inputs
		if m.Type.NumIn() != 2 {
			return nil, fmt.Errorf("one input structure expected in method %s", m.Name)
		}

		in := m.Type.In(1)
		if in.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("input must be a pointer in method %s", m.Name)
		}
		inT := in.Elem()
		if inT.Kind() != reflect.Struct {
			return nil, fmt.Errorf("input must be a pointer to struct in method %s", m.Name)
		}

		// check outputs
		if m.Type.NumOut() != 2 {
			return nil, fmt.Errorf("expected response and error as output in method %s", m.Name)
		}
		out := m.Type.Out(0)
		if out.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("output must be a pointer in method %s", m.Name)
		}
		outT := out.Elem()
		if outT.Kind() != reflect.Struct {
			return nil, fmt.Errorf("output must be a pointer to struct in method %s", m.Name)
		}
		out = m.Type.Out(1)
		if !out.Implements(errorInterface) {
			return nil, fmt.Errorf("method must return error type in method %s", m.Name)
		}

		if _, ok := rpc.methods[m.Name]; ok {
			return nil, fmt.Errorf("Method %s is already registered", m.Name)
		}
		rpc.methods[m.Name] = methodDetails{m.Func, inT, outT}
	}
	rpc.protocol = f
	return rpc, nil

}

func (rpc *RPCServer) Run() {
	for conn := range rpc.conns {
		go rpc.procConn(conn)
	}
}

func (rpc *RPCServer) procConn(conn RPCTransport) {
	prot := rpc.protocol()
	notifier := &RPCNotifier{make(chan *Packet)}
	prot.OnConnect(conn, notifier)
	for {
		var retPacket *Packet
		select {
		case packet := <-conn.Recv():
			retPacket = rpc.callMethod(prot, packet)

		case retPacket = <-notifier.wait():

		case err := <-conn.Closed():
			if err != nil {
				fmt.Printf("DEBUG: returning procConn() with err: %v\n", err) // FIXME
			}
			prot.OnDisconnect(err)
			return

		}

		err := conn.Send(retPacket)
		if err != nil {
			// logging error
			fmt.Printf("cant send: %s\n", err) //FIXME
			return
		}
	}
}

func (rpc *RPCServer) callMethod(p SessionProtocol, packet *Packet) *Packet {
	m, ok := rpc.methods[packet.Header.Method] //FIXME lock
	if !ok {
		return packet.Error(
			fmt.Errorf("no method %s found", packet.Header.Method),
		)
	}

	inV := reflect.New(m.inType)
	err := json.Unmarshal(packet.Body, inV.Interface())
	if err != nil {
		return packet.Error(err)
	}

	ret := m.funcVal.Call([]reflect.Value{reflect.ValueOf(p), inV})

	var buf []byte
	if !ret[1].IsNil() { // check error
		return packet.Error(ret[1].Interface().(error))
	}

	buf, err = json.Marshal(ret[0].Interface())
	if err != nil {
		// FIXME logging
		return packet.Error(err)
	}
	h := Header{
		MessageId: packet.Header.MessageId,
		Type:      PT_RESPONSE,
		Method:    packet.Header.Method,
	}
	return &Packet{Header: h, Body: buf}
}
