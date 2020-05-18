package filesystem

import (
	"bytes"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/source"
	encunicode "golang.org/x/text/encoding/unicode"
)

var utf16encoding = encunicode.UTF16(encunicode.LittleEndian, encunicode.IgnoreBOM)
var utf16encoder = utf16encoding.NewEncoder()
var utf16decoder = utf16encoding.NewDecoder()

type file struct {
	fullPath string
	content  []byte
	open     bool

	ls   source.Lines
	errs bool
}

func NewFile(fullPath string, content []byte) *file {
	return &file{fullPath: fullPath, content: content}
}

func (f *file) FullPath() string {
	return f.fullPath
}

func (f *file) Dir() string {
	return filepath.Dir(f.FullPath())
}

func (f *file) Filename() string {
	return filepath.Base(f.FullPath())
}

func (f *file) URI() string {
	return URIFromPath(f.fullPath)
}

func (f *file) Lines() source.Lines {
	if f.ls == nil {
		f.ls = source.MakeSourceLines(f.Filename(), f.content)
	}
	return f.ls
}

func (f *file) Text() []byte {
	return f.content
}

func (f *file) applyChange(change FileChange) error {
	// hcl pos column and line start from 1
	// such case, we regard it as full content change
	if change.Range().End.Column == 0 && change.Range().End.Line == 0 {
		f.content = []byte(change.Text())
		f.ls = nil
		return nil
	}
	b := &bytes.Buffer{}
	b.Grow(len(change.Text()) + len(f.content) - (change.Range().End.Byte - change.Range().Start.Byte))
	b.Write(f.content[:change.Range().Start.Byte])
	b.WriteString(change.Text())
	b.Write(f.content[change.Range().End.Byte:])

	f.content = b.Bytes()
	f.ls = nil
	return nil
}
