package filesystem

import (
	"io/ioutil"
	"log"
	"sync"

	"github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

type fsystem struct {
	mu sync.RWMutex

	logger *log.Logger
	dirs   map[string]*dir
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
		dirs:   make(map[string]*dir),
		logger: log.New(ioutil.Discard, "", 0),
	}
}

func (fs *fsystem) SetLogger(logger *log.Logger) {
	fs.logger = logger
}

func (fs *fsystem) Open(doc lsp.TextDocumentItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	u := URI(doc.URI)
	s := []byte(doc.Text)

	if !u.Valid() {
		return &InvalidURIErr{URI: u}
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
		return &FileNotOpenErr{u}
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
		return &FileNotOpenErr{u}
	}
	_, dn, fn := u.PathParts()
	delete(fs.dirs[dn].files, fn)
	return nil
}

func (fs *fsystem) URI(uri lsp.DocumentURI) URI {
	return URI(uri)
}

func (fs *fsystem) HclBlockAtDocPosition(params lsp.TextDocumentPositionParams) (*hcl.Block, hcl.Pos, error) {
	u := fs.URI(params.TextDocument.URI)
	f := fs.file(u)
	if f == nil || !f.open {
		return nil, hcl.Pos{}, &FileNotOpenErr{u}
	}

	fs.logger.Printf("Converting LSP position %#v into HCL", params.Position)

	hclPos, err := f.LspPosToHCLPos(params.Position)
	if err != nil {
		return nil, hcl.Pos{}, err
	}

	fs.logger.Printf("Finding HCL block at position %#v", hclPos)

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
