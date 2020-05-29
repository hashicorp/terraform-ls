package filesystem

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/source"
)

type File interface {
	FileHandler
	Text() []byte
	Lines() source.Lines
	Version() int
}

type FilePosition interface {
	FileHandler
	Position() hcl.Pos
}

type FileChange interface {
	Text() string
	Range() hcl.Range
}

type VersionedFileHandler interface {
	FileHandler
	Version() int
}

type FileHandler interface {
	URI() string
	FullPath() string
	Dir() string
	Filename() string
}

type FileChanges []FileChange

type Filesystem interface {
	Open(File) error
	Change(VersionedFileHandler, FileChanges) error
	Close(FileHandler) error
	GetFile(FileHandler) (File, error)
}
