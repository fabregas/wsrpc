package wsrpc

import (
	_ "fmt"
	"testing"
	"time"
)

func TestRPCBothSide(t *testing.T) {
	closech := make(chan struct{})
	notifch := make(chan *MyNotif)
	go ServeWSRPC(func() SessionProtocol { return &MyProtocol{} }, ":8080", "/test/wsrpc", &DummyLogger{LL_INFO}, closech)
	time.Sleep(100 * time.Millisecond)

	tr, err := NewWsConn("ws://127.0.0.1:8080/test/wsrpc", &DummyLogger{})
	if err != nil {
		t.Error(err)
		return
	}

	onNotifFunc := func(n interface{}, err error) {
		if err != nil {
			t.Error(err)
			return
		}

		switch notif := n.(type) {
		case *MyNotif:
			notifch <- notif
		default:
			t.Error("unexpected notification type")
		}
	}

	cli, err := NewRPCClient(tr, &MyProtocol{}, 1*time.Second, onNotifFunc, &DummyLogger{})
	if err != nil {
		t.Error(err)
		return
	}

	// check good scenario
	r := SomeReq{"Alice"}
	respI, err := cli.Call("MyMethod", &r)
	if err != nil {
		t.Error(err)
		return
	}
	resp := respI.(*SomeResp)
	if resp.IsBob != false {
		t.Error("unexpectee response")
		return
	}

	// check invalid method
	respI, err = cli.Call("MyMethodInvalid", &r)
	if err.Error() != "unknown method MyMethodInvalid" {
		t.Error(err)
		return
	}

	// check rpc method error
	respI, err = cli.Call("MyMethod", &SomeReq{})
	if err.Error() != "empty name!" {
		t.Error(err)
		return
	}

	// check invalid request type
	type SomeOther struct {
		Ohoho int
	}
	respI, err = cli.Call("MyMethod", &SomeOther{33})
	if err.Error() != "invalid request type, *SomeReq expected" {
		t.Error(err)
		return
	}
	rd, ok := respI.(*SomeResp)
	if rd != nil {
		t.Error("nil expected")
		return
	}
	if ok {
		t.Error("unepected cast")
		return
	}

	// check notification
	notif := <-notifch
	if notif.Msg != "hello, dude!" {
		t.Error(notif.Msg)
		return
	}
	// check timeout using sleep method
	_, err = cli.Call("MySleep", &r)
	if err != TimeoutError {
		t.Error(err)
		return
	}

	if cli.Closed() == true {
		t.Error("expected not closed cli conn")
		return
	}

	close(closech)
	time.Sleep(100 * time.Millisecond)

	_, err = cli.Call("MyMethod", &r)
	if err != ClosedConnError {
		t.Error(err)
		return
	}
	_, err = cli.Call("MyMethod", &r)
	if err != ClosedConnError {
		t.Error(err)
		return
	}

	if cli.Closed() == false {
		t.Error("expected closed cli conn")
		return
	}
}
