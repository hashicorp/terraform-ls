package filesystem

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/source"
)

type Document interface {
	DocumentHandler
	Text() ([]byte, error)
	Lines() source.Lines
	Version() int
}

type DocumentHandler interface {
	URI() string
	FullPath() string
	Dir() string
	Filename() string
}

type VersionedDocumentHandler interface {
	DocumentHandler
	Version() int
}

type DocumentChange interface {
	Text() string
	Range() hcl.Range
}

type DocumentChanges []DocumentChange

type DocumentStorage interface {
	// LS-specific methods
	CreateDocument(DocumentHandler, []byte) error
	CreateAndOpenDocument(DocumentHandler, []byte) error
	GetDocument(DocumentHandler) (Document, error)
	CloseAndRemoveDocument(DocumentHandler) error
	ChangeDocument(VersionedDocumentHandler, DocumentChanges) error
}
