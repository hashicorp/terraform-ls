package module

import (
	"container/heap"
	"sync"

	"github.com/hashicorp/terraform-ls/internal/document"
)

type moduleOpsQueue struct {
	q  *queue
	mu *sync.Mutex
}

func newModuleOpsQueue(ds DocumentStore) moduleOpsQueue {
	q := moduleOpsQueue{
		q: &queue{
			ops: make([]ModuleOperation, 0),
			ds:  ds,
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

func (q *moduleOpsQueue) DequeueAllModuleOps(modPath string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.q.Len() == 0 {
		return
	}

	for i, p := range q.q.ops {
		if p.ModulePath == modPath {
			q.q.ops = append(q.q.ops[:i], q.q.ops[:i+1]...)
		}
	}
}

func (q *moduleOpsQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.q.Len()
}

type queue struct {
	ops []ModuleOperation
	ds  DocumentStore
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

	aModHandle := document.DirHandleFromPath(aModOp.ModulePath)
	if hasOpenFiles, _ := q.ds.HasOpenDocuments(aModHandle); hasOpenFiles {
		leftOpen = 1
	}
	bModHandle := document.DirHandleFromPath(bModOp.ModulePath)
	if hasOpenFiles, _ := q.ds.HasOpenDocuments(bModHandle); hasOpenFiles {
		rightOpen = 1
	}

	return leftOpen > rightOpen
}
