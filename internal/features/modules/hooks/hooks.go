// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

// Package hooks enables the implementation of hooks for dynamic
// autocompletion. Hooks should be added to this package and
// registered via AppendCompletionHooks in completion_hooks.go.
package hooks

import (
	"log"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/registry"
)

type Hooks struct {
	ModStore       *state.ModuleStore
	RegistryClient registry.Client
	AlgoliaClient  *search.Client
	Logger         *log.Logger
}
