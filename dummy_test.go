package wsrpc

import (
	"fmt"
	"time"
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
	conn   *RPCConn

	Notifications struct {
		*MyNotif
	}
}

func (p *MyProtocol) OnConnect(conn *RPCConn) {
	conn.Notify(&MyNotif{"hello, dude!"})
	p.conn = conn
}
func (p *MyProtocol) OnDisconnect(err error) {
	//fmt.Println("ON DISCONNECT => ", err)
	p.closed <- true
}
func (p *MyProtocol) MyMethod(req *SomeReq) (*SomeResp, error) {
	if req.Name == "Bad" {
		return nil, fmt.Errorf("bad name!")
	}

	if req.Name == "Bob" {
		return &SomeResp{true}, nil
	} else {
		return &SomeResp{false}, nil
	}
}
func (p *MyProtocol) MySleep(req *SomeReq) (*SomeResp, error) {
	time.Sleep(2 * time.Second)
	return &SomeResp{false}, nil
}

func (p *MyProtocol) EmitInvalidNotification(req *SomeReq) (*SomeResp, error) {
	err := p.conn.Notify(&SomeResp{})
	return nil, err
}

// --------------------------------------------
// fake transport implementation

type FakeConn struct {
	in  chan *Packet
	out chan *Packet
}

func (c *FakeConn) Recv() (*Packet, error) {
	p := <-c.in
	if p == nil {
		return nil, fmt.Errorf("simulated close")
	}
	return p, nil
}
func (c *FakeConn) Send(p *Packet) error {
	p, _ = ParsePacket(p.Dump()) //simulate dump/parse in real scenario
	c.out <- p
	return nil
}

func (c *FakeConn) Close() error {
	c.in <- nil
	return nil
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
	}
}

// --------------------------------------------
// dummy logger implementation
const (
	LL_CRITICAL = iota
	LL_ERROR
	LL_WARNING
	LL_INFO
	LL_DEBUG
)

type DummyLogger struct {
	ll int
}

func (l *DummyLogger) Critical(args ...interface{}) {
	if l.ll < LL_CRITICAL {
		return
	}
	fmt.Printf("[CRITICAL] ")
	fmt.Println(args...)
}
func (l *DummyLogger) Criticalf(format string, args ...interface{}) {
	if l.ll < LL_CRITICAL {
		return
	}
	fmt.Printf("[CRITICAL] "+format+"\n", args...)
}

func (l *DummyLogger) Error(args ...interface{}) {
	if l.ll < LL_ERROR {
		return
	}
	fmt.Printf("[ERROR] ")
	fmt.Println(args...)
}
func (l *DummyLogger) Errorf(format string, args ...interface{}) {
	if l.ll < LL_ERROR {
		return
	}
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
func (l *DummyLogger) Warning(args ...interface{}) {
	if l.ll < LL_WARNING {
		return
	}
	fmt.Printf("[WARNING] ")
	fmt.Println(args...)
}
func (l *DummyLogger) Warningf(format string, args ...interface{}) {
	if l.ll < LL_WARNING {
		return
	}
	fmt.Printf("[WARNING] "+format+"\n", args...)
}
func (l *DummyLogger) Info(args ...interface{}) {
	if l.ll < LL_INFO {
		return
	}
	fmt.Printf("[INFO] ")
	fmt.Println(args...)
}
func (l *DummyLogger) Infof(format string, args ...interface{}) {
	if l.ll < LL_INFO {
		return
	}
	fmt.Printf("[INFO] "+format+"\n", args...)
}
func (l *DummyLogger) Debug(args ...interface{}) {
	if l.ll < LL_DEBUG {
		return
	}
	fmt.Printf("[DEBUG] ")
	fmt.Println(args...)
}
func (l *DummyLogger) Debugf(format string, args ...interface{}) {
	if l.ll < LL_DEBUG {
		return
	}
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}
