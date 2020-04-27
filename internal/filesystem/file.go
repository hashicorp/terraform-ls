package filesystem

import (
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

const uriPrefix = "file://"

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
	return uriPrefix + f.fullPath
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
	newBytes := []byte(change.Text())
	f.change(newBytes)

	return nil
}

func (f *file) change(s []byte) {
	f.content = s
	f.ls = nil
}
