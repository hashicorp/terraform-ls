package module

import (
	"container/heap"
	"sync"
)

type moduleOpsQueue struct {
	q  queue
	mu *sync.Mutex
}

func newModuleOpsQueue() moduleOpsQueue {
	q := moduleOpsQueue{
		q:  make(queue, 0),
		mu: &sync.Mutex{},
	}
	heap.Init(&q.q)
	return q
}

func (q *moduleOpsQueue) PushOp(op ModuleOperation) {
	q.mu.Lock()
	defer q.mu.Unlock()

	heap.Push(&q.q, op)

}

func (q *moduleOpsQueue) PopOp() ModuleOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := heap.Pop(&q.q)
	modOp := item.(ModuleOperation)
	return modOp
}

func (q *moduleOpsQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.q.Len()
}

func (q *moduleOpsQueue) Peek() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	item := q.q.Peek()
	return item
}

type queue []ModuleOperation

var _ heap.Interface = &queue{}

func (q *queue) Push(x interface{}) {
	modOp := x.(ModuleOperation)
	*q = append(*q, modOp)
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *queue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	*q = old[0 : n-1]
	return item
}

func (q queue) Peek() interface{} {
	n := len(q)
	return q[n-1]
}

func (q queue) Len() int {
	return len(q)
}

func (q queue) Less(i, j int) bool {
	return moduleOperationLess(q[i], q[j])
}

func moduleOperationLess(aModOp, bModOp ModuleOperation) bool {
	leftOpen, rightOpen := 0, 0

	if aModOp.Module.HasOpenFiles() {
		leftOpen = 1
	}
	if bModOp.Module.HasOpenFiles() {
		rightOpen = 1
	}

	return leftOpen > rightOpen
}
