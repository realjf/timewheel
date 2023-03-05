package timer

import (
	"sync"

	"github.com/realjf/spinlock"
)

const (
	TIME_NEAR_SHIFT  = 8
	TIME_NEAR        = (1 << TIME_NEAR_SHIFT)
	TIME_LEVEL_SHIFT = 6
	TIME_LEVEL       = (1 << TIME_LEVEL_SHIFT)
	TIME_NEAR_MASK   = (TIME_NEAR - 1)
	TIME_LEVEL_MASK  = (TIME_LEVEL - 1)
)

type Timer struct {
	Near         [TIME_NEAR]*LinkedList
	Timewheel    [4][TIME_LEVEL]*LinkedList
	lock         sync.Locker
	tick         uint32
	currentTime  uint64
	currentPoint uint64
}

func NewTimer() *Timer {
	return &Timer{
		Near:      [TIME_NEAR]*LinkedList{},
		Timewheel: [4][TIME_LEVEL]*LinkedList{},
		lock:      spinlock.NewSpinLock(),
	}
}

func (t *Timer) AddTimer(tick uint32, handler CallbackFunc, threadId int) *TimerNode {
	node := NewTimerNode()
	t.lock.Lock()
	defer t.lock.Unlock()

	node.Expire = tick + t.tick
	node.Callback = handler
	node.Id = threadId
	if tick <= 0 {
		node.Callback(node)
		return nil
	}
	t.addNode(node)
	return node
}

func (t *Timer) addNode(node *TimerNode) {
	var tick uint32 = node.Expire
	var currentTime uint32 = t.tick
	var msec uint32 = tick - currentTime
	if msec < TIME_NEAR {
		// [0, 0x100)
		t.link(t.Near[tick&TIME_NEAR_MASK], node)
	} else if msec < (1 << (TIME_NEAR_SHIFT + TIME_LEVEL_SHIFT)) {
		// [0x100, 0x4000)
		t.link(t.Timewheel[0][((tick>>TIME_NEAR_SHIFT)&TIME_LEVEL_MASK)], node)
	} else if msec < (1 << (TIME_NEAR_SHIFT + 2*TIME_LEVEL_SHIFT)) {
		// [0x4000, 0x100000)
		t.link(t.Timewheel[1][((tick>>(TIME_NEAR_SHIFT+TIME_LEVEL_SHIFT))&TIME_LEVEL_MASK)], node)
	} else if msec < (1 << (TIME_NEAR_SHIFT + 3*TIME_LEVEL_SHIFT)) {
		// [0x100000, 0x4000000)
		t.link(t.Timewheel[2][((tick>>(TIME_NEAR_SHIFT+2*TIME_LEVEL_SHIFT))&TIME_LEVEL_MASK)], node)
	} else {
		// [0x4000000, 0xffffffff]
		t.link(t.Timewheel[3][((tick>>(TIME_NEAR_SHIFT+3*TIME_LEVEL_SHIFT))&TIME_LEVEL_MASK)], node)
	}
}

func (t *Timer) link(linkList *LinkedList, node *TimerNode) {
	if linkList == nil {
		linkList = NewLinkedList()
	}
	node.Next = nil
	if linkList.Head == nil {
		linkList.Head = node
		linkList.Tail = node
	} else {
		linkList.Tail.Next = node
		linkList.Tail = node
	}
}

func (t *Timer) Start() {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.run()
	t.shift()
	t.run()
}

func (t *Timer) run() {
	var idx int = int(t.tick & TIME_NEAR_MASK)

	for t.Near[idx].Head.Next != nil {
		var current *TimerNode = t.linkClear(t.Near[idx])
		t.lock.Lock()
		t.dispatchList(current)
		t.lock.Unlock()
	}
}

func (t *Timer) dispatchList(node *TimerNode) {
	for node != nil {
		tmp := node
		node = node.Next
		if !tmp.Cancel {
			tmp.Callback(tmp)
		}
	}
}

func (t *Timer) shift() {
	var mask int = TIME_NEAR
	t.tick++
	var ct uint32 = t.tick
	if ct == 0 {
		t.moveList(3, 0)
	} else {
		// ct / 256
		var tick uint32 = ct >> TIME_NEAR_SHIFT
		var i int = 0
		// ct % 256 == 0
		for (ct & uint32(mask-1)) == 0 {
			// floor(ct/2^8) % 2^6
			var idx int = int(tick & TIME_LEVEL_MASK)
			if idx != 0 {
				t.moveList(i, idx)
				break
			}
			mask <<= TIME_LEVEL_SHIFT
			tick >>= TIME_LEVEL_SHIFT
			i++
		}
	}
}

func (t *Timer) moveList(level, idx int) {
	var current *TimerNode = t.linkClear(t.Timewheel[level][idx])
	for current != nil {
		tmp := current.Next
		t.addNode(current)
		current = tmp
	}
}

func (t *Timer) linkClear(linkList *LinkedList) *TimerNode {
	ret := linkList.Head.Next
	linkList.Head.Next = nil
	linkList.Tail = linkList.Head
	return ret
}

func (t *Timer) Stop() {

}

func (t *Timer) DelTimer(node *TimerNode) {
	node.Cancel = true
}
