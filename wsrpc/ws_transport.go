package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func init() {
	rand.Seed(time.Now().Unix())
}

type WsTransport struct {
	in     chan *Packet
	out    chan *Packet
	closed chan error

	conn      *websocket.Conn
	pongWait  time.Duration
	writeWait time.Duration
}

func NewWsTransport(c *websocket.Conn) *WsTransport {
	t := &WsTransport{
		in:        make(chan *Packet),
		out:       make(chan *Packet),
		closed:    make(chan error),
		conn:      c,
		pongWait:  60 * time.Second, //FIXME
		writeWait: 10 * time.Second, //FIXME
	}
	go t.readLoop()
	go t.writeLoop()
	return t
}

func (t *WsTransport) Recv() <-chan *Packet {
	return t.in
}
func (t *WsTransport) Send(p *Packet) error {
	t.out <- p
	//	return t.conn.WriteMessage(websocket.BinaryMessage, p.Dump())
	return nil
}

func (t *WsTransport) Close() error {
	//	t.closed <- nil
	//	return nil
	t.conn.WriteMessage(websocket.CloseMessage, []byte{})
	return t.conn.Close()
}

func (t *WsTransport) Closed() <-chan error {
	return t.closed
}

func (t *WsTransport) readLoop() {
	pongHandler := func(string) error {
		t.conn.SetReadDeadline(time.Now().Add(t.pongWait))
		return nil
	}
	t.conn.SetPongHandler(pongHandler)
	t.conn.SetReadDeadline(time.Now().Add(t.pongWait))
	defer func() {
		t.conn.Close()
	}()

	for {
		_, raw, err := t.conn.ReadMessage()
		//fmt.Println(">> ", mtype)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
				t.closed <- nil
			} else {
				t.closed <- err
			}
			break
		}

		p, err := ParsePacket(raw)
		if err != nil {
			fmt.Printf("Parse packet error: %s\n", err)
			t.conn.Close()
			t.closed <- err
			break
		}
		t.in <- p

	}
}

func (t *WsTransport) writeLoop() {
	rp := time.Duration((rand.Intn(20) + 70))
	pingPeriod := (t.pongWait * rp) / 100
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		t.conn.Close()
	}()
	for {
		select {
		case p := <-t.out:
			t.conn.SetWriteDeadline(time.Now().Add(t.writeWait))
			err := t.conn.WriteMessage(websocket.BinaryMessage, p.Dump())
			if err != nil {
				fmt.Printf("write packet error: %s\n", err) //FIXME
				return
			}
		case <-ticker.C:
			t.conn.SetWriteDeadline(time.Now().Add(t.writeWait))
			if err := t.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

type WsHandler struct {
	conns    chan RPCTransport
	upgrader websocket.Upgrader
}

func NewWsHandler() *WsHandler {
	return &WsHandler{
		make(chan RPCTransport),
		websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (h *WsHandler) Connections() <-chan RPCTransport {
	return h.conns
}

func (h *WsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err) //FIXME logging
		return
	}
	h.conns <- NewWsTransport(conn)
}

type WsClient struct {
	transport *WsTransport
}

func NewWsClient(url string) (*WsClient, error) {
	dialer := &websocket.Dialer{}
	conn, resp, err := dialer.Dial(url, http.Header{})
	if err != nil {
		fmt.Println(resp) // FIXME
		return nil, err
	}

	return &WsClient{NewWsTransport(conn)}, nil
}
