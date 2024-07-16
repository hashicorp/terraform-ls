// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
)

// LoadStackComponentSources will trigger parsing the local terraform modules for a stack in the ModulesFeature
func LoadStackComponentSources(ctx context.Context, stackStore *state.StackStore, bus *eventbus.EventBus, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// iterate over each component in the stack and find local terraform modules
	for _, component := range record.Meta.Components {
		if component.Source == "" {
			// no source recorded, skip
			continue
		}

		var fullPath string
		// detect if component.Source is a local module
		switch component.SourceAddr.(type) {
		case module.LocalSourceAddr:
			fullPath = filepath.Join(stackPath, filepath.FromSlash(component.Source))
		case tfaddr.Module:
			continue
		case module.RemoteSourceAddr:
			continue
		default:
			// Unknown source address, we can't resolve the path
			continue
		}

		if fullPath == "" {
			// Unknown source address, we can't resolve the path
			continue
		}

		dh := document.DirHandleFromPath(fullPath)

		// notify the event bus that a new Component with a
		// local modules has been opened
		bus.DidOpen(eventbus.DidOpenEvent{
			Context:    ctx,
			Dir:        dh,
			LanguageID: lsp.Terraform.String(),
		})
	}

	return nil
}
