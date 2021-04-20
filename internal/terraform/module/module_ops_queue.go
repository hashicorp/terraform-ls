package module

import (
	"container/heap"
	"sync"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

type moduleOpsQueue struct {
	q  *queue
	mu *sync.Mutex
}

func newModuleOpsQueue(fs filesystem.Filesystem) moduleOpsQueue {
	q := moduleOpsQueue{
		q: &queue{
			ops: make([]ModuleOperation, 0),
			fs:  fs,
		},
		mu: &sync.Mutex{},
	}
	heap.Init(q.q)
	return q
}

func (q *moduleOpsQueue) PushOp(op ModuleOperation) {
	q.mu.Lock()
	defer q.mu.Unlock()

	heap.Push(q.q, op)
}

func (q *moduleOpsQueue) PopOp() (ModuleOperation, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.q.Len() == 0 {
		return ModuleOperation{}, false
	}

	item := heap.Pop(q.q)
	modOp := item.(ModuleOperation)
	return modOp, true
}

func (q *moduleOpsQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.q.Len()
}

type queue struct {
	ops []ModuleOperation
	fs  filesystem.Filesystem
}

var _ heap.Interface = &queue{}

func (q *queue) Push(x interface{}) {
	modOp := x.(ModuleOperation)
	q.ops = append(q.ops, modOp)
}

func (q *queue) Swap(i, j int) {
	q.ops[i], q.ops[j] = q.ops[j], q.ops[i]
}

func (q *queue) Pop() interface{} {
	old := q.ops
	n := len(old)
	item := old[n-1]
	q.ops = old[0 : n-1]
	return item
}

func (q *queue) Len() int {
	return len(q.ops)
}

func (q *queue) Less(i, j int) bool {
	return q.moduleOperationLess(q.ops[i], q.ops[j])
}

func (q *queue) moduleOperationLess(aModOp, bModOp ModuleOperation) bool {
	leftOpen, rightOpen := 0, 0

	if hasOpenFiles, _ := q.fs.HasOpenFiles(aModOp.ModulePath); hasOpenFiles {
		leftOpen = 1
	}
	if hasOpenFiles, _ := q.fs.HasOpenFiles(bModOp.ModulePath); hasOpenFiles {
		rightOpen = 1
	}

	return leftOpen > rightOpen
}
