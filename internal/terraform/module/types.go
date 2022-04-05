package module

import (
	"context"
	"io/fs"
	"log"

	"github.com/hashicorp/terraform-ls/internal/document"
)

type Watcher interface {
	Start(context.Context) error
	Stop() error
	SetLogger(*log.Logger)
	AddModule(string) error
	RemoveModule(string) error
	IsModuleWatched(string) bool
}

type ReadOnlyFS interface {
	fs.FS
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Stat(name string) (fs.FileInfo, error)
}

type DocumentStore interface {
	HasOpenDocuments(dirHandle document.DirHandle) (bool, error)
}
