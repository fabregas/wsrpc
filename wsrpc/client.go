package wsrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"
)

type RPCClient struct {
	conn          RPCTransport
	flow          *FlowController
	notifications chan *Packet
	closedFlag    int32

	methods map[string]methodDetails
}

func NewRPCClient(conn RPCTransport, p SessionProtocol, timeout time.Duration) (*RPCClient, error) {
	cli := &RPCClient{
		conn:          conn,
		flow:          NewFlowController(timeout),
		notifications: make(chan *Packet, 100),
	}
	methods, err := parseSessionProtocol(p)
	cli.methods = methods

	if err != nil {
		return nil, err
	}
	go cli.loop()
	return cli, nil
}

func (cli *RPCClient) Call(method string, request interface{}) (interface{}, error) {
	md, ok := cli.methods[method]
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

func (cli *RPCClient) RawNotifications() <-chan *Packet {
	return cli.notifications
}

func (cli *RPCClient) onNotif(packet *Packet) {
	select {
	case cli.notifications <- packet:
	default:
		// drop notifiaction if nobody recvs it
	}
}

func (cli *RPCClient) loop() {
	for {
		select {
		case packet := <-cli.conn.Recv():
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
				fmt.Printf("ERROR: unexpected packet type <%s>\n", packet.Header.Type) //FIXME logging
			}

		case err := <-cli.conn.Closed():
			if err != nil {
				fmt.Println("cli.loop() closed with error: ", err)
			}
			atomic.StoreInt32(&cli.closedFlag, 1)
			return
		}
	}

}
