package priorityqueue

import (
	"container/heap"
	"testing"
)

type Queue struct {
	Name     string
	Priority int
}

func TestPriorityQueue(t *testing.T) {
	intHeap := &GenericList[int]{
		LessFunc: func(a, b int) bool { return a < b },
	}
	heap.Init(intHeap)
	heap.Push(intHeap, 3)
	heap.Push(intHeap, 1)
	heap.Push(intHeap, 2)

	if heap.Pop(intHeap) != 1 {
		t.Errorf("expected 1")
	}
	if heap.Pop(intHeap) != 2 {
		t.Errorf("expected 2")
	}
	if heap.Pop(intHeap) != 3 {
		t.Errorf("expected 3")
	}
	if intHeap.Len() != 0 {
		t.Errorf("expected empty heap")
	}
}

func TestQueue(t *testing.T) {
	queue := &GenericList[Queue]{
		LessFunc: func(a, b Queue) bool { return a.Priority < b.Priority },
	}

	heap.Init(queue)
	heap.Push(queue, Queue{Name: "low", Priority: 1})
	heap.Push(queue, Queue{Name: "medium", Priority: 5})
	heap.Push(queue, Queue{Name: "high", Priority: 10})

	if heap.Pop(queue) != (Queue{Name: "low", Priority: 1}) {
		t.Errorf("expected low priority queue item")
	}
	if heap.Pop(queue) != (Queue{Name: "medium", Priority: 5}) {
		t.Errorf("expected medium priority queue item")
	}
	if heap.Pop(queue) != (Queue{Name: "high", Priority: 10}) {
		t.Errorf("expected high priority queue item")
	}
	if queue.Len() != 0 {
		t.Errorf("expected empty queue")
	}
}
