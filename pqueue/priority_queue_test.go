package pqueue

import (
	"container/heap"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func equal(t *testing.T, act, exp interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
			filepath.Base(file), line, exp, act)
		t.FailNow()
	}
}

func TestPriorityQueue(t *testing.T) {
	cases := map[string]struct {
		F func()
	}{
		"push_and_pop": {
			F: func() {
				c := 100
				pq := New(c)

				for i := 0; i < c+1; i++ {
					heap.Push(&pq, &Item{Value: i, Priority: int64(i)})
				}

				t.Logf("queue len: %d", pq.Len())
				t.Logf("queue cap: %d", cap(pq))
				assert.Equal(t, pq.Len(), int(c+1))
				assert.Equal(t, cap(pq), int(c*2))

				for i := 0; i < c+1; i++ {
					item := heap.Pop(&pq)
					assert.Equal(t, item.(*Item).Value.(int), i)
				}

				assert.Equal(t, cap(pq), c/4)
			},
		},
		"unsorted_insert": {
			F: func() {
				c := 100
				pq := New(c)
				ints := make([]int, 0, c)

				for i := 0; i < c; i++ {
					v := rand.Int()
					ints = append(ints, v)
					heap.Push(&pq, &Item{Value: i, Priority: int64(v)})
				}

				t.Logf("queue len: %d", pq.Len())
				t.Logf("queue cap: %d", cap(pq))
				assert.Equal(t, pq.Len(), int(c))
				assert.Equal(t, cap(pq), int(c))

				sort.Ints(ints)

				for i := 0; i < c; i++ {
					item, _ := pq.PeekAndShift(int64(ints[len(ints)-1]))
					assert.Equal(t, item.Priority, int64(ints[i]))
				}
			},
		},
		"remove": {
			F: func() {
				c := 100
				pq := New(c)

				for i := 0; i < c; i++ {
					v := rand.Int()
					heap.Push(&pq, &Item{Value: "test", Priority: int64(v)})
				}

				for i := 0; i < 10; i++ {
					heap.Remove(&pq, rand.Intn((c-1)-i))
				}

				lastPriority := heap.Pop(&pq).(*Item).Priority
				for i := 0; i < (c - 10 - 1); i++ {
					item := heap.Pop(&pq)
					assert.Equal(t, lastPriority < item.(*Item).Priority, true)
					lastPriority = item.(*Item).Priority
				}
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.F()
		})
	}
}
