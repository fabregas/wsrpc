package wsrpc

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dummyOnNotifFunc(n interface{}, err error) {
}

func BenchmarkWsRPCServer(b *testing.B) {
	closech := make(chan struct{})
	log := &DummyLogger{LL_ERROR}
	go ServeWSRPC(func() SessionProtocol { return &MyProtocol{} }, ":8080", "/test/wsrpc", log, closech)
	time.Sleep(1 * time.Second)
	/////////////

	cli, err := NewWsConn("ws://127.0.0.1:8080/test/wsrpc", &DummyLogger{})
	if err != nil {
		panic(err)
	}
	cli.Recv() //recv notification
	p := NewPacket(PT_REQUEST, "MyMethod", []byte("{\"name\":\"Bob\"}"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cli.Send(p)
		cli.Recv()
	}
	b.StopTimer()
	cli.Close()
	close(closech)
}

func BenchmarkRawWsServer(b *testing.B) {
	wsfunc := func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			panic(err)
			return
		}
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				//fmt.Println(err)
				return
			}
			if err := conn.WriteMessage(messageType, p); err != nil {
				fmt.Println(err)
				panic(err)
				return
			}
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test/ws", wsfunc)
	s := http.Server{Handler: mux, Addr: ":8081"}
	go s.ListenAndServe()
	time.Sleep(1 * time.Second)

	/////////////

	dialer := &websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://127.0.0.1:8081/test/ws", http.Header{})
	//	conn.EnableWriteCompression(false)
	if err != nil {
		panic(err)
	}

	msg := []byte("some message for echo testing raw WS socket")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := conn.WriteMessage(websocket.BinaryMessage, msg)
		if err != nil {
			panic(err)
		}
		_, _, err = conn.ReadMessage()
		if err != nil {
			panic(err)
			return
		}
	}
	b.StopTimer()
	s.Close()
}

func BenchmarkRawTcpServer(b *testing.B) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", "127.0.0.1:7777")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				return
			}
			// Handle connections in a new goroutine.
			go handleRequest(conn)
		}
	}()

	time.Sleep(1 * time.Second)

	/////////////

	conn, _ := net.Dial("tcp", "127.0.0.1:7777")
	msg := []byte("some message for echo testing raw tcp socket")
	buf := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.Write(msg)
		conn.Read(buf)
	}
	b.StopTimer()

	conn.Close()
	l.Close()
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	for {
		reqLen, err := conn.Read(buf)
		if err != nil {
			break
		}
		// Send a response back to person contacting us.
		conn.Write(buf[:reqLen])
	}
	// Close the connection when you're done with it.
	conn.Close()
}
