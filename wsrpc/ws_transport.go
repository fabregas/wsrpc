package wsrpc

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ClosedConnError = fmt.Errorf("closed connection")
)

func init() {
	rand.Seed(time.Now().Unix())
}

type WsTransport struct {
	in       chan *Packet
	closedCh chan error
	wlock    sync.Mutex

	conn       *websocket.Conn
	pingTicker *time.Ticker
	pongWait   time.Duration
	writeWait  time.Duration
}

func NewWsTransport(c *websocket.Conn) *WsTransport {
	pongWait := 60 * time.Second  //FIXME
	writeWait := 10 * time.Second //FIXME

	rp := time.Duration((rand.Intn(20) + 70))
	pingPeriod := (pongWait * rp) / 100
	ticker := time.NewTicker(pingPeriod)

	t := &WsTransport{
		in:         make(chan *Packet),
		closedCh:   make(chan error),
		conn:       c,
		pingTicker: ticker,
		pongWait:   pongWait,
		writeWait:  writeWait,
	}
	go t.readLoop()
	go t.pingLoop()
	return t
}

func (t *WsTransport) Recv() <-chan *Packet {
	return t.in
}

func (t *WsTransport) Send(p *Packet) error {
	buf := p.Dump()

	t.wlock.Lock()
	t.conn.SetWriteDeadline(time.Now().Add(t.writeWait))
	err := t.conn.WriteMessage(websocket.BinaryMessage, buf)
	t.wlock.Unlock()

	if err != nil {
		if err == websocket.ErrCloseSent {
			t.conn.Close()
			return ClosedConnError
		}
		return err
	}
	return nil
}

func (t *WsTransport) Close() error {
	t.wlock.Lock()
	t.conn.WriteMessage(websocket.CloseMessage, []byte{})
	t.wlock.Unlock()

	return t.conn.Close()
}

func (t *WsTransport) Closed() <-chan error {
	return t.closedCh
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
		t.pingTicker.Stop()
		close(t.in)
	}()

	for {
		_, raw, err := t.conn.ReadMessage()
		//fmt.Println(">> ", mtype)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
				t.closedCh <- nil
			} else {
				t.closedCh <- err
			}
			break
		}

		p, err := ParsePacket(raw)
		if err != nil {
			fmt.Printf("Parse packet error: %s\n", err)
			t.closedCh <- err
			break
		}
		t.in <- p
	}
}

func (t *WsTransport) pingLoop() {
	for _ = range t.pingTicker.C {
		t.wlock.Lock()
		t.conn.SetWriteDeadline(time.Now().Add(t.writeWait))
		err := t.conn.WriteMessage(websocket.PingMessage, []byte{})
		t.wlock.Unlock()
		if err != nil {
			t.conn.Close()
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
