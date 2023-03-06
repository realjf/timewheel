package timewheel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/realjf/timewheel"
)

type EveryScheduler struct {
	Interval time.Duration
}

func (s *EveryScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}

func TestExample_scheduleTimer(t *testing.T) {
	t.Run("scheduleTimer", func(t *testing.T) {
		tw := timewheel.NewTimeWheel(time.Millisecond, 20)
		tw.Start()
		defer tw.Stop()

		exitC := make(chan time.Time)
		timer := tw.ScheduleFunc(&EveryScheduler{time.Second}, func() {
			fmt.Println("The timer fires")
			exitC <- time.Now().Local()
		})

		<-exitC
		<-exitC

		// We need to stop the timer since it will be restarted again and again.
		for !timer.Stop() {
		}
	})

	// Output:
	// The timer fires
	// The timer fires
}
