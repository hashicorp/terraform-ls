package filesystem

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

type fsystem struct {
	mu   sync.RWMutex
	dirs map[string]*dir
}

type Filesystem interface {
	Open(lsp.TextDocumentItem) error
	Change(lsp.VersionedTextDocumentIdentifier, []lsp.TextDocumentContentChangeEvent) error
	Close(lsp.TextDocumentIdentifier) error
	URI(lsp.DocumentURI) URI
	HclBlockAtDocPosition(lsp.TextDocumentPositionParams) (*hcl.Block, hcl.Pos, error)
}

func NewFilesystem() *fsystem {
	return &fsystem{
		dirs: make(map[string]*dir),
	}
}

func (fs *fsystem) Open(doc lsp.TextDocumentItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	u := URI(doc.URI)
	s := []byte(doc.Text)

	if !u.Valid() {
		return fmt.Errorf("invalid URL to open")
	}

	fullName, dn, fn := u.PathParts()
	d, ok := fs.dirs[dn]
	if !ok {
		d = newDir()
		fs.dirs[dn] = d
	}
	f, ok := d.files[fn]
	if !ok {
		f = NewFile(fullName, s)
	}
	f.open = true
	d.files[fn] = f
	return nil
}

func (fs *fsystem) Change(doc lsp.VersionedTextDocumentIdentifier, changes []lsp.TextDocumentContentChangeEvent) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	u := URI(doc.URI)

	f := fs.file(u)
	if f == nil || !f.open {
		return fmt.Errorf("file %q is not open", u)
	}
	for _, change := range changes {
		f.applyChange(change)
	}
	return nil
}

func (fs *fsystem) Close(doc lsp.TextDocumentIdentifier) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	u := URI(doc.URI)

	f := fs.file(u)
	if f == nil || !f.open {
		return fmt.Errorf("file %q is not open", u)
	}
	_, dn, fn := u.PathParts()
	delete(fs.dirs[dn].files, fn)
	return nil
}

func (fs *fsystem) URI(uri lsp.DocumentURI) URI {
	return URI(uri)
}

func (fs *fsystem) HclBlockAtDocPosition(params lsp.TextDocumentPositionParams) (*hcl.Block, hcl.Pos, error) {
	u := URI(params.TextDocument.URI)
	f := fs.file(u)

	hclPos := f.LspPosToHCLPos(params.Position)

	block, err := f.HclBlockAtPos(hclPos)

	return block, hclPos, err
}

func (fs *fsystem) file(u URI) *file {
	if !u.Valid() {
		return nil
	}
	_, dn, fn := u.PathParts()
	d, ok := fs.dirs[dn]
	if !ok {
		return nil
	}
	return d.files[fn]
}
