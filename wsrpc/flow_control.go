package wsrpc

import (
	"fmt"
	"sync"
	"time"
)

var (
	TimeoutError = fmt.Errorf("waiter timeouted")
)

type Waiter struct {
	sync.RWMutex
	data *Packet
	err  error
	ttl  time.Time
}

func NewWaiter(ttl time.Time) *Waiter {
	w := Waiter{ttl: ttl}
	w.Lock() // lock waiter for write
	return &w
}

func (w *Waiter) Timeouted() bool {
	return time.Now().After(w.ttl)
}

func (w *Waiter) setData(data *Packet) {
	w.data = data
	w.Unlock() // waiter ready for read
}

func (w *Waiter) setError(err error) {
	w.err = err
	w.Unlock() // waiter ready for read
}

func (w *Waiter) Wait() (*Packet, error) {
	w.RLock()
	defer w.RUnlock()
	return w.data, w.err
}

type FlowController struct {
	sync.Mutex
	waiters map[string]*Waiter
	timeout time.Duration
}

func NewFlowController(timeout time.Duration) *FlowController {
	fc := &FlowController{
		waiters: make(map[string]*Waiter),
		timeout: timeout,
	}
	go fc.checkTimeouts()
	return fc
}

func (fc *FlowController) checkTimeouts() {
	for _ = range time.Tick(fc.timeout / 3) {
		fc.Lock()
		for key, w := range fc.waiters {
			if w.Timeouted() {
				delete(fc.waiters, key)
				w.setError(TimeoutError)
			}
		}
		fc.Unlock()
	}
}

func (fc *FlowController) NewWaiter(mid string) *Waiter {
	ttl := time.Now().Add(fc.timeout)
	rw := NewWaiter(ttl)
	fc.Lock()
	fc.waiters[mid] = rw
	fc.Unlock()
	return rw
}

func (fc *FlowController) GetWaiter(mid string) *Waiter {
	fc.Lock()
	rw := fc.waiters[mid]
	delete(fc.waiters, mid)
	fc.Unlock()
	return rw
}
