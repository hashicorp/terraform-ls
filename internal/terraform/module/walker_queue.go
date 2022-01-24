package module

import (
	"container/heap"

	"github.com/hashicorp/terraform-ls/internal/document"
)

type walkerQueue struct {
	paths []string

	ds DocumentStore
}

var _ heap.Interface = &walkerQueue{}

func newWalkerQueue(ds DocumentStore) *walkerQueue {
	wq := &walkerQueue{
		paths: make([]string, 0),
		ds:    ds,
	}
	heap.Init(wq)
	return wq
}

func (q *walkerQueue) Push(x interface{}) {
	path := x.(string)

	if q.pathIsEnqueued(path) {
		// avoid duplicate entries
		return
	}

	q.paths = append(q.paths, path)
}

func (q *walkerQueue) pathIsEnqueued(path string) bool {
	for _, p := range q.paths {
		if p == path {
			return true
		}
	}
	return false
}

func (q *walkerQueue) RemoveFromQueue(path string) {
	for i, p := range q.paths {
		if p == path {
			q.paths = append(q.paths[:i], q.paths[i+1:]...)
		}
	}
}

func (q *walkerQueue) Swap(i, j int) {
	q.paths[i], q.paths[j] = q.paths[j], q.paths[i]
}

func (q *walkerQueue) Pop() interface{} {
	old := q.paths
	n := len(old)
	item := old[n-1]
	q.paths = old[0 : n-1]
	return item
}

func (q *walkerQueue) Len() int {
	return len(q.paths)
}

func (q *walkerQueue) Less(i, j int) bool {
	return q.moduleOperationLess(q.paths[i], q.paths[j])
}

func (q *walkerQueue) moduleOperationLess(leftModPath, rightModPath string) bool {
	leftOpen, rightOpen := 0, 0

	leftMod := document.DirHandleFromPath(leftModPath)
	if hasOpenFiles, _ := q.ds.HasOpenDocuments(leftMod); hasOpenFiles {
		leftOpen = 1
	}

	rightMod := document.DirHandleFromPath(rightModPath)
	if hasOpenFiles, _ := q.ds.HasOpenDocuments(rightMod); hasOpenFiles {
		rightOpen = 1
	}

	return leftOpen > rightOpen
}
