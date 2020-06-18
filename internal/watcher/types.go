package watcher

import (
	"log"
)

type TrackedFile interface {
	Path() string
	Sha256Sum() string
}

type Watcher interface {
	Start() error
	Stop() error
	SetLogger(logger *log.Logger)
	AddPath(path string) error
	AddPaths(paths []string) error
	AddChangeHook(f ChangeHook)
}

type ChangeHook func(file TrackedFile) error
