package delayqueue

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"

	"github.com/realjf/timewheel/pqueue"
)

type DelayQueue struct {
	C chan interface{}

	mu sync.Mutex
	pq pqueue.PriorityQueue

	sleeping int32
	wakeupC  chan struct{}
}

func New(size int) *DelayQueue {
	return &DelayQueue{
		C:       make(chan interface{}),
		pq:      pqueue.New(size),
		wakeupC: make(chan struct{}),
	}
}

func (dq *DelayQueue) Push(elem interface{}, expiration int64) {
	item := &pqueue.Item{Value: elem, Priority: expiration}

	dq.mu.Lock()
	heap.Push(&dq.pq, item)
	index := item.Index
	dq.mu.Unlock()

	if index == 0 {
		if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
			dq.wakeupC <- struct{}{}
		}
	}
}

func (dq *DelayQueue) Poll(exitC chan struct{}, nowF func() int64) {
	for {
		now := nowF()

		dq.mu.Lock()
		item, delta := dq.pq.PeekAndShift(now)
		if item == nil {
			atomic.StoreInt32(&dq.sleeping, 1)
		}
		dq.mu.Unlock()

		if item == nil {
			if delta == 0 { // No items left
				select {
				case <-dq.wakeupC:
					// Wait until a new item is added
					continue
				case <-exitC:
					goto exit
				}
			} else if delta > 0 { // At least one item is pending
				select {
				case <-dq.wakeupC:
					//
					continue
				case <-time.After(time.Duration(delta) * time.Millisecond):

					// Reset the sleeping state since there's no need to receive from wakeupC
					if atomic.SwapInt32(&dq.sleeping, 0) == 0 {
						<-dq.wakeupC
					}
					continue
				case <-exitC:
					goto exit
				}
			}
		}

		select {
		case dq.C <- item.Value:
			// The expired element has been sent out successfully.
		case <-exitC:
			goto exit
		}
	}
exit:
	atomic.StoreInt32(&dq.sleeping, 0)
}
