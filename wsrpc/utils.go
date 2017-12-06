package wsrpc

import (
	"net/http"
)

func ServeWSRPC(sfunc NewSessionFunc, addr string, path string, closeCh chan struct{}) {
	wsh := NewWsHandler()
	srv, err := NewRPCServer(wsh.Connections(), sfunc)
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
