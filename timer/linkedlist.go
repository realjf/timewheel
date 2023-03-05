package timer

type LinkedList struct {
	Head *TimerNode
	Tail *TimerNode
}

func NewLinkedList() *LinkedList {
	return &LinkedList{}
}
