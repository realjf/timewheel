package delayqueue

import (
	"testing"
	"time"

	"github.com/realjf/timewheel/util"
)

func TestDelayQueue(t *testing.T) {
	cases := map[string]struct {
		tick time.Duration
	}{
		"test_10ms": {
			tick: time.Duration(10 * time.Millisecond),
		},
		"test_20ms": {
			tick: time.Duration(20 * time.Millisecond),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := 100
			dq := New(c)
			tick := tc.tick
			tickMs := int64(tick / time.Millisecond)

			exitC := make(chan struct{})
			wg := util.NewWaitGroupWrapper()

			wg.Wrap(func() {
				dq.Poll(exitC, func() int64 {
					return util.TimeToMs(time.Now().Local())
				})
			})

			wg.Wrap(func() {
				for {
					select {
					case elem := <-dq.C:
						b := elem.(int)
						t.Logf("pop: %d", b)
					case <-exitC:
						t.Logf("done")
						return
					}
				}
			})

			startMs := util.TimeToMs(time.Now().Local())
			currentTime := util.Truncate(startMs, tickMs)
			go func(s int64) {
				for i := 0; i < c; i++ {
					t.Logf("%d", s+tickMs)
					dq.Push(i, s+tickMs)
					s += tickMs
				}
			}(currentTime)

			go func() {
				time.Sleep(5 * time.Second)
				close(exitC)
			}()

			wg.Wait()

			t.Log("exit!!!")
		})
	}
}
