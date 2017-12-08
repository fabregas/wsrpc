package wsrpc

import (
	"net/http"
	"time"
)

func ServeWSRPC(sfunc NewSessionFunc, addr string, path string, log Logger, closeCh chan struct{}) {
	wsh := NewWsHandler(log)
	srv, err := NewRPCServer(wsh.Connections(), sfunc, log)
	if err != nil {
		panic(err)
	}
	go srv.Run()

	mux := http.NewServeMux()
	mux.Handle(path, wsh)
	s := http.Server{Handler: mux, Addr: addr}
	go s.ListenAndServe()

	<-closeCh

	s.Close()
	srv.Close()
}

func ClientWSRPC(p SessionProtocol, url string, timeout time.Duration, onNotifFunc OnNotificationFunc, log Logger) (*RPCClient, error) {
	tr, err := NewWsConn(url, log)
	if err != nil {
		return nil, err
	}

	return NewRPCClient(tr, p, timeout, onNotifFunc, log)
}
