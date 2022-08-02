// Package hooks enables the implementation of hooks for dynamic
// autocompletion. Hooks should be added to this package and
// registered via AppendCompletionHooks in completion_hooks.go.
package hooks

import "github.com/hashicorp/terraform-ls/internal/state"

type Hooks struct {
	ModStore *state.ModuleStore
}
