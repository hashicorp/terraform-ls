// Package hooks enables the implementation of hooks for dynamic
// autocompletion. Hooks should be added to this package and
// registered in AppendCompletionHooks in completion_hooks.go.
//
// A hook must have the following signature:
//  func (h *Hooks) Name(ctx context.Context, value cty.Value) ([]decoder.Candidate, error)
// It receives the current value of the attribute and must return
// a list of completion candidates.
//
// All hooks have access to path, filename and pos via context:
//  path, ok := decoder.PathFromContext(ctx)
//  filename, ok := decoder.FilenameFromContext(ctx)
//  pos, ok := decoder.PosFromContext(ctx)
package hooks

import "github.com/hashicorp/terraform-ls/internal/state"

type Hooks struct {
	ModStore *state.ModuleStore
}
