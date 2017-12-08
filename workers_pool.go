package wsrpc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

type job struct {
	prot   SessionProtocol
	packet *Packet
	respCh chan *Packet
}

type workersPool struct {
	jobs        chan job
	wg          sync.WaitGroup
	log         Logger
	protDetails *protocolDetails
}

func (wp *workersPool) worker() {
	wp.log.Debug("starting new worker")
	defer wp.wg.Done()

	for j := range wp.jobs {
		j.respCh <- wp.callMethod(j.prot, j.packet)
	}
	wp.log.Debug("worker stopped")
}

func (wp *workersPool) Process(j job) {
	select {
	case wp.jobs <- j:
	default:
		wp.wg.Add(1)
		go wp.worker()
		wp.jobs <- j
	}
}

func (wp *workersPool) Close() {
	close(wp.jobs)
	wp.wg.Wait()
}

func (wp *workersPool) callMethod(p SessionProtocol, packet *Packet) *Packet {
	m, ok := wp.protDetails.methods[packet.Header.Method] //FIXME lock (?)
	if !ok {
		return packet.Error(
			fmt.Errorf("no method %s found", packet.Header.Method),
		)
	}

	inV := reflect.New(m.inType)
	err := json.Unmarshal(packet.Body, inV.Interface())
	if err != nil {
		return packet.Error(err)
	}

	ret := m.funcVal.Call([]reflect.Value{reflect.ValueOf(p), inV})

	var buf []byte
	if !ret[1].IsNil() { // check error
		return packet.Error(ret[1].Interface().(error))
	}

	buf, err = json.Marshal(ret[0].Interface())
	if err != nil {
		return packet.Error(err)
	}

	h := Header{
		MessageId: packet.Header.MessageId,
		Type:      PT_RESPONSE,
		Method:    packet.Header.Method,
	}
	return &Packet{Header: h, Body: buf}
}
