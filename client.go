package wsrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"
)

type OnNotificationFunc func(interface{}, error)

type RPCClient struct {
	conn          RPCTransport
	flow          *FlowController
	notifications chan *Packet
	closedFlag    int32

	protDetails *protocolDetails
	onNotifFunc OnNotificationFunc

	log Logger
}

func NewRPCClient(
	conn RPCTransport,
	p SessionProtocol,
	timeout time.Duration,
	onNotifFunc OnNotificationFunc,
	log Logger,
) (*RPCClient, error) {
	if conn == nil {
		return nil, fmt.Errorf("Nil RPCTransport passed")
	}
	cli := &RPCClient{
		conn:          conn,
		flow:          NewFlowController(timeout),
		notifications: make(chan *Packet, 100),
		onNotifFunc:   onNotifFunc,
		log:           log,
	}
	pdetails, err := parseSessionProtocol(p)
	if err != nil {
		return nil, err
	}
	cli.protDetails = pdetails
	go cli.loop()
	go cli.notifLoop()
	return cli, nil
}

func (cli *RPCClient) Call(method string, request interface{}) (interface{}, error) {
	md, ok := cli.protDetails.methods[method]
	if !ok {
		return nil, fmt.Errorf("unknown method %s", method)
	}

	// check request type
	if md.inType != reflect.TypeOf(request).Elem() {
		return nil, fmt.Errorf("invalid request type, *%s expected", md.inType.Name())
	}

	// create request message
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	reqPacket := NewPacket(PT_REQUEST, method, reqBody)

	// set new response waiter
	rid := reqPacket.Id()
	rw := cli.flow.NewWaiter(rid)

	// send request to server
	err = cli.conn.Send(reqPacket)
	if err != nil {
		cli.flow.GetWaiter(rid)
		return nil, err
	}

	respPacket, err := rw.Wait()
	if err != nil {
		return nil, err
	}

	// unmarshal result
	outV := reflect.New(md.outType)
	err = json.Unmarshal(respPacket.Body, outV.Interface())
	if err != nil {
		return nil, err
	}
	return outV.Interface(), nil
}

func (cli *RPCClient) Closed() bool {
	return atomic.LoadInt32(&cli.closedFlag) == 1
}
func (cli *RPCClient) Close() error {
	return cli.conn.Close()
}

func (cli *RPCClient) notifLoop() {
	for packet := range cli.notifications {
		if cli.onNotifFunc == nil {
			// just ignore notification
			continue
		}
		vt, ok := cli.protDetails.notifications[packet.Header.Method]
		if !ok {
			cli.onNotifFunc(nil, fmt.Errorf("unexpected notification %s", packet.Header.Method))
			return
		}
		val := reflect.New(vt)
		err := json.Unmarshal(packet.Body, val.Interface())

		cli.onNotifFunc(val.Interface(), err)
	}
}

func (cli *RPCClient) onNotif(packet *Packet) {
	select {
	case cli.notifications <- packet:
	default:
		// drop notifiaction if nobody recvs it
		cli.log.Warning("notification dropped bcs no free room in channel")
	}
}

func (cli *RPCClient) loop() {
	cli.log.Debug("rpc client read loop started")
	for {
		packet, err := cli.conn.Recv()

		if err != nil {
			cli.log.Debugf("[cli.loop] closed with error: %s", err.Error())
			close(cli.notifications)
			atomic.StoreInt32(&cli.closedFlag, 1)
			return
		}

		switch packet.Header.Type {
		case PT_ERROR:
			rw := cli.flow.GetWaiter(packet.Id())
			if rw != nil {
				rw.setError(errors.New(string(packet.Body)))
			}

		case PT_RESPONSE:
			rw := cli.flow.GetWaiter(packet.Id())
			if rw != nil {
				rw.setData(packet)
			}

		case PT_NOTIFICATION:
			cli.onNotif(packet)

		default:
			cli.log.Errorf("[cli.loop] unexpected packet type <%s>", packet.Header.Type)
		}
	}

}
