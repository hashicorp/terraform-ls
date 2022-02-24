package module

import (
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func SyncWalker(fs ReadOnlyFS, ds DocumentStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore, js job.JobStore, tfExec exec.ExecutorFactory) *Walker {
	w := NewWalker(fs, ds, ms, pss, js, tfExec)
	w.sync = true
	return w
}
