package protocol

import (
	"../../../"

	"fmt"
	"io"
)

type SumReq struct {
	A int
	B int
}

type SumResp struct {
	Sum int
}

type ExampleNotif struct {
	Msg   string
	Descr string
}

type SumProtocol struct {
	Notifications struct {
		*ExampleNotif
	}
}

func (p *SumProtocol) OnConnect(closer io.Closer, notifier *wsrpc.RPCNotifier) {
	fmt.Println("Some client connected ...")
	err := notifier.Notify(
		&ExampleNotif{
			"hello, dude!",
			"you can sum any two natural numbers using this API",
		},
	)
	if err != nil {
		fmt.Println("send notification error: ", err)
	}
}
func (p *SumProtocol) OnDisconnect(err error) {
	fmt.Printf("Client is disconnected (err=%v)\n", err)
}

func (p *SumProtocol) Sum(req *SumReq) (*SumResp, error) {
	if req.A <= 0 || req.B <= 0 {
		return nil, fmt.Errorf("A and B must be natual numbers!")
	}
	return &SumResp{req.A + req.B}, nil
}
