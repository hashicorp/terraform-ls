package filesystem

import (
	"bytes"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/source"
	"github.com/hashicorp/terraform-ls/internal/uri"
	"github.com/spf13/afero"
)

type document struct {
	meta *documentMetadata
	fs   afero.Fs
}

func (d *document) Text() ([]byte, error) {
	return afero.ReadFile(d.fs, d.meta.dh.FullPath())
}

func (d *document) FullPath() string {
	return d.meta.dh.FullPath()
}

func (d *document) Dir() string {
	return filepath.Dir(d.meta.dh.FullPath())
}

func (d *document) Filename() string {
	return filepath.Base(d.meta.dh.FullPath())
}

func (d *document) URI() string {
	return uri.FromPath(d.meta.dh.FullPath())
}

func (d *document) Lines() source.Lines {
	return d.meta.Lines()
}

func (d *document) Version() int {
	return d.meta.Version()
}

func (d *document) IsOpen() bool {
	return d.meta.IsOpen()
}

func (d *document) Equal(doc *document) bool {
	if d.URI() != doc.URI() {
		return false
	}
	if d.IsOpen() != doc.IsOpen() {
		return false
	}
	if d.Version() != doc.Version() {
		return false
	}

	leftB, err := d.Text()
	if err != nil {
		return false
	}
	rightB, err := doc.Text()
	if err != nil {
		return false
	}
	return bytes.Equal(leftB, rightB)
}
