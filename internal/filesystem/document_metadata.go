package filesystem

import (
	"sync"

	"github.com/hashicorp/terraform-ls/internal/source"
)

type documentMetadata struct {
	dh DocumentHandler

	mu      *sync.RWMutex
	isOpen  bool
	version int
	lines   source.Lines
}

func NewDocumentMetadata(dh DocumentHandler, content []byte) *documentMetadata {
	return &documentMetadata{
		dh:    dh,
		mu:    &sync.RWMutex{},
		lines: source.MakeSourceLines(dh.Filename(), content),
	}
}

func (d *documentMetadata) setOpen(isOpen bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.isOpen = isOpen
}

func (d *documentMetadata) setVersion(version int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.version = version
}

func (d *documentMetadata) updateLines(content []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lines = source.MakeSourceLines(d.dh.Filename(), content)
}

func (d *documentMetadata) Lines() source.Lines {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lines
}

func (d *documentMetadata) Version() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.version
}

func (d *documentMetadata) IsOpen() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isOpen
}
