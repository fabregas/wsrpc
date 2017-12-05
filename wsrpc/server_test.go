package main

import (
	"fmt"
	"io"
	"testing"
)

// --------------------------------------------
// simple protocol implementation

type SomeReq struct {
	Name string
}

type SomeResp struct {
	IsBob bool
}

type MyNotif struct {
	Msg string
}

type MyProtocol struct {
	closed chan bool
}

func (p *MyProtocol) OnConnect(closer io.Closer, notifier *RPCNotifier) {
	//fmt.Println("ON CONNECT =>", closer, notifier)
	notifier.Notify(&MyNotif{"hello, dude!"})
}
func (p *MyProtocol) OnDisconnect(err error) {
	//fmt.Println("ON DISCONNECT => ", err)
	p.closed <- true
}
func (p *MyProtocol) MyMethod(req *SomeReq) (*SomeResp, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("empty name!")
	}

	if req.Name == "Bob" {
		return &SomeResp{true}, nil
	} else {
		return &SomeResp{false}, nil
	}
}

// --------------------------------------------
// fake transport implementation

type FakeConn struct {
	in     chan *Packet
	out    chan *Packet
	closed chan error
}

func (c *FakeConn) Recv() <-chan *Packet {
	return c.in
}
func (c *FakeConn) Send(p *Packet) error {
	p, _ = ParsePacket(p.Dump()) //simulate dump/parse in real scenario
	c.out <- p
	return nil
}

func (c *FakeConn) Close() error {
	c.closed <- nil
	return nil
}
func (c *FakeConn) Closed() <-chan error {
	return c.closed
}
func (c *FakeConn) simulateReq() *Packet {
	p := NewPacket(PT_REQUEST, "MyMethod", []byte("{\"name\":\"Bob\"}"))
	p, _ = ParsePacket(p.Dump()) //simulate dump/parse in real scenario
	c.in <- p
	return p
}

func NewFakeConn() *FakeConn {
	return &FakeConn{
		make(chan *Packet),
		make(chan *Packet, 1),
		make(chan error),
	}
}

func TestAbstractRPC(t *testing.T) {
	conns := make(chan RPCTransport)
	closeCh := make(chan bool)

	srv, err := NewRPCServer(conns, func() SessionProtocol { return &MyProtocol{closeCh} })
	if err != nil {
		t.Error(err)
		return
	}
	go srv.Run()

	conn := NewFakeConn()
	conns <- conn
	// check notification
	p := <-conn.out
	if len(p.Header.MessageId) != 16 {
		t.Error("invalid notif msgid")
		return
	}
	if p.Header.Type != PT_NOTIFICATION {
		t.Error("invalid notif type")
		return
	}
	if string(p.Body) != "{\"Msg\":\"hello, dude!\"}" {
		t.Error("invalid notif body")
		return
	}

	// check response
	req := conn.simulateReq()
	p = <-conn.out
	if string(p.Header.MessageId) != string(req.Header.MessageId) {
		t.Error("invalid msg id")
		return
	}
	if p.Header.Type != PT_RESPONSE {
		t.Error("invalid resp type")
		return
	}
	if p.Header.Method != "MyMethod" {
		t.Error("invalid resp method")
		return
	}
	if string(p.Body) != "{\"IsBob\":true}" {
		fmt.Println(string(p.Body))
		t.Error("invalid resp body ")
		return
	}

	// check method error
	req = NewPacket(PT_REQUEST, "MyMethod", []byte("{\"name\":\"\"}"))
	conn.in <- req
	p = <-conn.out
	if string(p.Header.MessageId) != string(req.Header.MessageId) {
		t.Error("invalid msg id")
		return
	}
	if p.Header.Type != PT_ERROR {
		t.Error("invalid err type")
		return
	}
	if string(p.Body) != "empty name!" {
		t.Error("invalid err body")
		return
	}

	// check fail
	req = NewPacket(PT_REQUEST, "UnknMethod", []byte("{\"name\":\"Bob\"}"))
	conn.in <- req
	p = <-conn.out
	if string(p.Header.MessageId) != string(req.Header.MessageId) {
		t.Error("invalid msg id")
		return
	}
	if p.Header.Type != PT_ERROR {
		t.Error("invalid err type")
		return
	}
	if string(p.Body) != "no method UnknMethod found" {
		t.Error("invalid err body")
		return
	}

	conn.Close()
	<-closeCh
}

func BenchmarkAbstractRPCServer(b *testing.B) {
	conns := make(chan RPCTransport)

	srv, err := NewRPCServer(conns, func() SessionProtocol { return &MyProtocol{nil} })
	if err != nil {
		b.Error(err)
		return
	}
	go srv.Run()

	conn := NewFakeConn()
	conns <- conn
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.simulateReq()
		<-conn.out
	}
}
