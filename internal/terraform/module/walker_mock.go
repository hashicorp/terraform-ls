package module

import "github.com/hashicorp/terraform-ls/internal/filesystem"

func SyncWalker(fs filesystem.Filesystem, modMgr ModuleManager) *Walker {
	w := NewWalker(fs, modMgr)
	w.sync = true
	return w
}
