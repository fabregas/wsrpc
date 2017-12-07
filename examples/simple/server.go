package main

import (
	"../../wsrpc"
	"./protocol"

	"os"
	"os/signal"

	"github.com/op/go-logging"
)

func main() {
	log := logging.MustGetLogger("example-server")
	logging.SetLevel(logging.INFO, "")

	closech := make(chan struct{})
	// start RPC server in separate goroutine
	go wsrpc.ServeWSRPC(func() wsrpc.SessionProtocol { return &protocol.SumProtocol{} }, ":8080", "/test/wsrpc", log, closech)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	close(closech)
}
