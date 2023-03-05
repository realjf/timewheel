package timewheel

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/realjf/timewheel/delayqueue"
	"github.com/realjf/timewheel/util"
)

type TimeWheel struct {
	tick      int64
	wheelSize int64

	interval    int64
	currentTime int64
	buckets     []*bucket
	queue       *delayqueue.DelayQueue

	overflowWheel unsafe.Pointer

	exitC     chan struct{}
	waitGroup util.WaitGroupWrapper
}

func NewTimeWheel(tick time.Duration, wheelSize int64) *TimeWheel {
	tickMs := int64(tick / time.Millisecond)
	if tickMs <= 0 {
		panic(errors.New("tick must be greater than or equal to 1ms"))
	}

	startMs := util.TimeToMs(time.Now().Local())

	return newTimeWheel(
		tickMs,
		wheelSize,
		startMs,
		delayqueue.New(int(wheelSize)),
	)
}

func newTimeWheel(tickMs int64, wheelSize int64, startMs int64, queue *delayqueue.DelayQueue) *TimeWheel {
	buckets := make([]*bucket, wheelSize)
	for i := range buckets {
		buckets[i] = newBucket()
	}
	return &TimeWheel{
		tick:        tickMs,
		wheelSize:   wheelSize,
		currentTime: util.Truncate(startMs, tickMs),
		interval:    tickMs * wheelSize,
		buckets:     buckets,
		queue:       queue,
		exitC:       make(chan struct{}),
	}
}

func (tw *TimeWheel) add(t *Timer) bool {
	currentTime := atomic.LoadInt64(&tw.currentTime)
	if t.expiration < currentTime+tw.tick {
		// Already expired
		return false
	} else if t.expiration < currentTime+tw.interval {
		// Put it into its own bucket
		virtualID := t.expiration / tw.tick
		b := tw.buckets[virtualID%tw.wheelSize]
		b.Add(t)

		// Set the bucket expiration time
		if b.SetExpiration(virtualID * tw.tick) {
			tw.queue.Push(b, b.Expiration())
		}

		return true
	} else {
		// Out of the interval. Put it into the overflow wheel
		overflowWheel := atomic.LoadPointer(&tw.overflowWheel)
		if overflowWheel == nil {
			atomic.CompareAndSwapPointer(&tw.overflowWheel, nil, unsafe.Pointer(newTimeWheel(tw.interval, tw.wheelSize, currentTime, tw.queue)))
			overflowWheel = atomic.LoadPointer(&tw.overflowWheel)
		}
		return (*TimeWheel)(overflowWheel).add(t)
	}
}

func (tw *TimeWheel) addOrRun(t *Timer) {
	if !tw.add(t) {
		// Already expired

		go t.callback()
	}
}

func (tw *TimeWheel) advanceClock(expiration int64) {
	currentTime := atomic.LoadInt64(&tw.currentTime)
	if expiration >= currentTime+tw.tick {
		currentTime = util.Truncate(expiration, tw.tick)
		atomic.StoreInt64(&tw.currentTime, currentTime)

		// Try to advance the clock of the overflow wheel if present
		overflowWheel := atomic.LoadPointer(&tw.overflowWheel)
		if overflowWheel != nil {
			(*TimeWheel)(overflowWheel).advanceClock(currentTime)
		}
	}
}

// Start starts the current time wheel
func (tw *TimeWheel) Start() {
	tw.waitGroup.Wrap(func() {
		tw.queue.Poll(tw.exitC, func() int64 {
			return util.TimeToMs(time.Now().Local())
		})
	})

	tw.waitGroup.Wrap(func() {
		for {
			select {
			case elem := <-tw.queue.C:
				b := elem.(*bucket)
				tw.advanceClock(b.Expiration())
				b.Flush(tw.addOrRun)
			case <-tw.exitC:
				return
			}
		}
	})
}

// Stop stops the current time wheel
func (tw *TimeWheel) Stop() {
	close(tw.exitC)
	tw.waitGroup.Wait()
}

func (tw *TimeWheel) AfterFunc(d time.Duration, f func()) *Timer {
	t := &Timer{
		expiration: util.TimeToMs(time.Now().Local().Add(d)),
		callback:   f,
	}
	tw.addOrRun(t)
	return t
}

type Scheduler interface {
	Next(time.Time) time.Time
}

func (tw *TimeWheel) ScheduleFunc(s Scheduler, f func()) (t *Timer) {
	expiration := s.Next(time.Now().UTC())
	if expiration.IsZero() {
		// No time is scheduled, return nil.
		return
	}

	t = &Timer{
		expiration: util.TimeToMs(expiration),
		callback: func() {
			// Schedule the task to execute at the next time if possible.
			expiration := s.Next(util.MsToTime(t.expiration))
			if !expiration.IsZero() {
				t.expiration = util.TimeToMs(expiration)
				tw.addOrRun(t)
			}

			// Actually execute the task.
			f()
		},
	}
	tw.addOrRun(t)

	return
}
