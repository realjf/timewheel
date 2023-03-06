package timewheel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/realjf/timewheel"
)

func TestExample_startTimer(t *testing.T) {
	t.Run("start_timer", func(t *testing.T) {
		tw := timewheel.NewTimeWheel(time.Millisecond, 20)
		tw.Start()
		defer tw.Stop()

		exitC := make(chan time.Time, 1)
		tw.AfterFunc(time.Second, func() {
			fmt.Println("The timer fires")
			exitC <- time.Now().Local()
		})

		<-exitC
	})

	// Output:
	// The timer fires
}

func TestExample_stopTimer(t *testing.T) {
	t.Run("stop_timer", func(t *testing.T) {
		tw := timewheel.NewTimeWheel(time.Millisecond, 20)
		tw.Start()
		defer tw.Stop()

		timer := tw.AfterFunc(time.Second, func() {
			fmt.Println("The timer fires")
		})

		<-time.After(900 * time.Millisecond)
		// Stop the timer before it fires
		timer.Stop()
	})

	// Output:
	//
}
