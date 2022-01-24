package module

import (
	"io/fs"
)

func SyncWalker(fs fs.StatFS, ds DocumentStore, modMgr ModuleManager) *Walker {
	w := NewWalker(fs, ds, modMgr)
	w.sync = true
	return w
}
