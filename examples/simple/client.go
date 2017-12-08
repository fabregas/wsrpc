package main

import (
	"../.."
	"./protocol"

	"fmt"
	"time"

	"github.com/op/go-logging"
)

func onNotifFunc(n interface{}, err error) {
	if err != nil {
		fmt.Printf("error while receiving notification: %s\n", err.Error())
		return
	}

	switch notif := n.(type) {
	case *protocol.ExampleNotif:
		fmt.Printf("notification from server: msg='%s', descr='%s'\n", notif.Msg, notif.Descr)
	default:
		fmt.Printf("unexpected notification type (%+v)\n", n)
	}
}

func main() {
	log := logging.MustGetLogger("example-client")
	logging.SetLevel(logging.INFO, "")

	cli, err := wsrpc.ClientWSRPC(&protocol.SumProtocol{}, "ws://127.0.0.1:8080/test/wsrpc", 5*time.Second, onNotifFunc, log)
	if err != nil {
		panic(err)
	}

	resp, err := cli.Call("Sum", &protocol.SumReq{12, 44})
	if err != nil {
		panic(err)
	}
	fmt.Println("12 + 44 =", resp.(*protocol.SumResp).Sum)

	cli.Close()
}
