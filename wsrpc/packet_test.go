package wsrpc

import (
	"strings"
	"testing"
)

func TestPacket(t *testing.T) {
	p := NewPacket(PT_REQUEST, "MyTestMethod", []byte("some body ;)"))

	if len(p.Header.MessageId) != 16 {
		t.Error("invalid message id")
		return
	}
	if p.Header.Method != "MyTestMethod" {
		t.Error("invalid method")
		return
	}
	if p.Header.Type != PT_REQUEST {
		t.Error("invalid type")
		return
	}
	if string(p.Body) != "some body ;)" {
		t.Error("invalid body")
		return
	}

	dumped := p.Dump()
	pp, err := ParsePacket(dumped)
	if err != nil {
		t.Error("invalid packet")
		return
	}

	if string(p.Header.MessageId) != string(pp.Header.MessageId) {
		t.Error("invalid parsed message id")
		return
	}
	if p.Header.Method != pp.Header.Method {
		t.Error("invalid parsed method")
		return
	}
	if p.Header.Type != pp.Header.Type {
		t.Error("invalid parsed type")
		return
	}
	if string(p.Body) != string(pp.Body) {
		t.Error("invalid parsed body")
		return
	}

	pRepr := p.String()
	if !strings.Contains(pRepr, "<id=") {
		t.Error(pRepr)
	}
	if !strings.Contains(pRepr, "type=REQ, method=MyTestMethod>[some body ;)]") {
		t.Error(pRepr)
	}
	//just for coverage
	NewPacket(PT_RESPONSE, "-", []byte("-")).String()
	NewPacket(PT_NOTIFICATION, "-", []byte("-")).String()
	NewPacket(PT_ERROR, "-", []byte("-")).String()
	NewPacket(88, "-", []byte("-")).String()

}

func BenchmarkPacketDumpParse(b *testing.B) {
	p := NewPacket(PT_NOTIFICATION, "MyNotif", []byte("some notify ;)"))
	for i := 0; i < b.N; i++ {
		ParsePacket(p.Dump())
	}

}
