package util

import (
	"sync"
	"time"
)

func Truncate(x, m int64) int64 {
	if m <= 0 {
		return x
	}
	return x - x%m
}

func TimeToMs(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func MsToTime(t int64) time.Time {
	return time.Unix(0, t*int64(time.Millisecond)).Local()
}

func Get10Ms() time.Duration {
	return time.Duration(int64(time.Now().UnixNano() / int64(time.Millisecond) / 10))
}

type WaitGroupWrapper struct {
	sync.WaitGroup
}

func NewWaitGroupWrapper() *WaitGroupWrapper {
	return &WaitGroupWrapper{}
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	go func() {
		defer func() {
			w.Done()
		}()
		cb()
	}()
}
