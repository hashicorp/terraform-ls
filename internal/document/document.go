package document

import (
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-ls/internal/source"
)

type Document struct {
	Dir      DirHandle
	Filename string

	ModTime    time.Time
	LanguageID string
	Version    int

	// Text contains the document body stored as bytes.
	// It originally comes as string from the client via LSP
	// but bytes are accepted by HCL and io/fs APIs, hence preferred.
	Text []byte

	// Lines contains Text separated into lines to enable byte offset
	// computation for any position-based operations within HCL, such as
	// completion, hover, semantic token based highlighting, etc.
	// and to aid in calculating diff when formatting document.
	// LSP positions contain just line+column but hcl.Pos requires offset.
	Lines source.Lines
}

func (doc *Document) FullPath() string {
	return filepath.Join(doc.Dir.Path(), doc.Filename)
}

func (d *Document) Copy() *Document {
	return &Document{
		Dir:        DirHandle{URI: d.Dir.URI},
		Filename:   d.Filename,
		ModTime:    d.ModTime,
		LanguageID: d.LanguageID,
		Version:    d.Version,
		Text:       d.Text,
		Lines:      d.Lines.Copy(),
	}
}
