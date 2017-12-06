package wsrpc

import (
	"fmt"

	"github.com/satori/go.uuid"
)

// packet types
const (
	PT_REQUEST      = uint8(1)
	PT_RESPONSE     = uint8(2)
	PT_NOTIFICATION = uint8(3)
	PT_ERROR        = uint8(66)
)

func printableType(t uint8) string {
	switch t {
	case PT_REQUEST:
		return "REQ"
	case PT_RESPONSE:
		return "RESP"
	case PT_NOTIFICATION:
		return "NOTIF"
	case PT_ERROR:
		return "ERR"
	default:
		return "UNKNOWN"
	}
}

type Header struct {
	MessageId []byte
	Method    string
	Type      uint8
}

type Packet struct {
	Header
	Body []byte
}

func NewPacket(ptype uint8, method string, body []byte) *Packet {
	return &Packet{
		Header: Header{
			MessageId: uuid.NewV1().Bytes(),
			Type:      ptype,
			Method:    method,
		},
		Body: body,
	}
}

func (p *Packet) Id() string {
	mid, _ := uuid.FromBytes(p.Header.MessageId)
	return mid.String()
}

func (p *Packet) String() string {
	mid, _ := uuid.FromBytes(p.Header.MessageId)
	return fmt.Sprintf(
		"<id=%s, type=%s, method=%s>[%s]",
		mid, printableType(p.Header.Type), p.Header.Method,
		string(p.Body),
	)
}

// Error returns error packet as a response on current packet
func (p *Packet) Error(err error) *Packet {
	h := Header{MessageId: p.Header.MessageId, Type: PT_ERROR}
	return &Packet{Header: h, Body: []byte(err.Error())}
}

func (p *Packet) Dump() []byte {
	// message_id + type + method_len + method + body
	mlen := len(p.Header.Method)
	rlen := 18 + mlen + len(p.Body)
	r := make([]byte, rlen)
	copy(r, p.Header.MessageId)
	r[16] = byte(p.Header.Type)
	r[17] = byte(mlen)
	copy(r[18:18+mlen], []byte(p.Header.Method))
	copy(r[18+mlen:], p.Body)
	return r
}

/*
func (p *Packet) DumpTo(buf []byte) []byte {
	mlen := len(p.Header.Method)
	rlen := 18 + mlen + len(p.Body)

	if cap(buf) < rlen {
		newcap := 2 * cap(buf)
		if newcap < rlen {
			newcap = rlen
		}
		buf = make([]byte, newcap)
	}
	buf = buf[:0]

	buf = append(buf, p.Header.MessageId...)
	buf = append(buf, byte(p.Header.Type), byte(mlen))
	buf = append(buf, []byte(p.Header.Method)...)
	buf = append(buf, p.Body...)
	return buf
}
*/

func ParsePacket(raw []byte) (*Packet, error) {
	p := &Packet{}
	if len(raw) < 20 {
		return nil, fmt.Errorf("invalid packet size")
	}
	p.Header.MessageId = raw[:16]
	p.Header.Type = uint8(raw[16])
	mlen := uint8(raw[17])
	if len(raw) < int(18+mlen+1) {
		return nil, fmt.Errorf("invalid packet")
	}
	p.Header.Method = string(raw[18 : 18+mlen])
	p.Body = raw[18+mlen:]
	return p, nil
}
