# wsrpc
[![Build Status](https://travis-ci.org/fabregas/wsrpc.svg?branch=master)](https://travis-ci.org/fabregas/wsrpc)
[![Coverage Status](https://coveralls.io/repos/github/fabregas/wsrpc/badge.svg?branch=master)](https://coveralls.io/github/fabregas/wsrpc?branch=master)

`wsrpc` is a Go package which allows you to construct session based RPC over websocket connection (client and server sides).

## Usage

In order to start, go get this repository:

```golang
go get github.com/fabregas/wsrpc
```

### Session protocol declaration example

```go
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
```

Full client/server example see in [examples/simple](https://github.com/fabregas/wsrpc/tree/master/examples/simple) directory.
