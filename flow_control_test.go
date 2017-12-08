package wsrpc

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkWaiters(b *testing.B) {
	fc := NewFlowController(5 * time.Second)
	midCh := make(chan string, 1000)
	waitersCh := make(chan *Waiter, 1000)
	finishCh := make(chan int)
	go func() {
		i := 0
		for mid := range midCh {
			fc.GetWaiter(mid).setData(nil)
			i += 1
		}
		finishCh <- i
	}()
	go func() {
		i := 0
		for w := range waitersCh {
			w.Wait()
			//w.Wait()
			i += 1
		}
		finishCh <- i
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mid := fmt.Sprintf("test-%d", i)
		w := fc.NewWaiter(mid)
		waitersCh <- w
		midCh <- mid
	}
	close(midCh)
	close(waitersCh)
}
