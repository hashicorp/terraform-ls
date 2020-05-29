package filesystem

import (
	"bytes"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
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

	version int
	ls      source.Lines
	errs    bool
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

func (f *file) Version() int {
	return f.version
}

func (f *file) SetVersion(version int) {
	f.version = version
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
	// if the range is regarded as nil, we regard it as full content change
	if rangeIsNil(change.Range()) {
		f.change([]byte(change.Text()))
		return nil
	}
	b := &bytes.Buffer{}
	b.Grow(len(f.content) + diffLen(change))
	b.Write(f.content[:change.Range().Start.Byte])
	b.WriteString(change.Text())
	b.Write(f.content[change.Range().End.Byte:])

	f.change(b.Bytes())
	return nil
}

func (f *file) change(s []byte) {
	f.content = s
	f.ls = nil
}

// HCL column and line indexes start from 1, therefore if the any index
// contains 0, we assume it is an undefined range
func rangeIsNil(r hcl.Range) bool {
	return r.End.Column == 0 && r.End.Line == 0
}

func diffLen(change FileChange) int {
	rangeLen := change.Range().End.Byte - change.Range().Start.Byte
	return len(change.Text()) - rangeLen
}
