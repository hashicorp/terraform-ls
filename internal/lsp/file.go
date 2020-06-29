package lsp

import (
	"github.com/hashicorp/terraform-ls/internal/source"
	lsp "github.com/sourcegraph/go-lsp"
)

type File interface {
	URI() string
	FullPath() string
	Dir() string
	Filename() string
	Lines() source.Lines
}

type file struct {
	fh      *fileHandler
	ls      source.Lines
	text    []byte
	version int
}

func (f *file) URI() string {
	return f.fh.URI()
}

func (f *file) FullPath() string {
	return f.fh.FullPath()
}

func (f *file) Dir() string {
	return f.fh.Dir()
}

func (f *file) Filename() string {
	return f.fh.Filename()
}

func (f *file) Text() []byte {
	return f.text
}

func (f *file) Lines() source.Lines {
	return f.lines()
}

func (f *file) lines() source.Lines {
	if f.ls == nil {
		f.ls = source.MakeSourceLines(f.fh.Filename(), f.text)
	}
	return f.ls
}

func (f *file) Version() int {
	return f.version
}

func FileFromDocumentItem(doc lsp.TextDocumentItem) *file {
	return &file{
		fh:      FileHandlerFromDocumentURI(doc.URI),
		text:    []byte(doc.Text),
		version: doc.Version,
	}
}
