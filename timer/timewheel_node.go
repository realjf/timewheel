package timer

import "sync"

type CallbackFunc func(node *TimerNode)

type TimerNode struct {
	Next     *TimerNode   `json:"next"`     // pointer to the next task
	Expire   uint32       `json:"expire"`   // tick + expire
	Callback CallbackFunc `json:"callback"` // callback function
	Cancel   bool         `json:"cancel"`   // cancel timer
	Id       int          `json:"id"`
	lock     sync.Locker
}

func NewTimerNode() *TimerNode {
	return &TimerNode{}
}
