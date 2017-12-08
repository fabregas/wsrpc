package wsrpc

import (
	"fmt"
	"testing"
)

func TestAbstractRPC(t *testing.T) {
	conns := make(chan RPCTransport)
	closeCh := make(chan bool)

	srv, err := NewRPCServer(conns, func() SessionProtocol { return &MyProtocol{closed: closeCh} }, &DummyLogger{LL_INFO})
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
	req = NewPacket(PT_REQUEST, "MyMethod", []byte("{\"name\":\"Bad\"}"))
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
	if string(p.Body) != "bad name!" {
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

	// emit invalid notification from server
	req = NewPacket(PT_REQUEST, "EmitInvalidNotification", []byte("{\"name\":\"Bob\"}"))
	conn.in <- req
	p = <-conn.out
	if p.Header.Type != PT_ERROR {
		t.Error("invalid err type")
		return
	}
	if string(p.Body) != "Notification *wsrpc.SomeResp is not declared in protocol" {
		t.Error(string(p.Body))
		return
	}

	conn.Close()
	<-closeCh
}

func BenchmarkAbstractRPCServer(b *testing.B) {
	conns := make(chan RPCTransport)

	srv, err := NewRPCServer(conns, func() SessionProtocol { return &MyProtocol{closed: nil} }, &DummyLogger{})
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
