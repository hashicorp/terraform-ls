package module

import "github.com/hashicorp/terraform-ls/internal/filesystem"

func SyncWalker(fs filesystem.Filesystem, modMgr ModuleManager, srv Server) *Walker {
	w := NewWalker(fs, modMgr, srv)
	w.sync = true
	return w
}
