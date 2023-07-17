// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/progress"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (h *CmdHandler) EarlyValidationHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	dirUri, ok := args.GetString("uri")
	if !ok || dirUri == "" {
		return nil, fmt.Errorf("%w: expected module uri argument to be set", jrpc2.InvalidParams.Err())
	}

	if !uri.IsURIValid(dirUri) {
		return nil, fmt.Errorf("URI %q is not valid", dirUri)
	}

	dirHandle := document.DirHandleFromURI(dirUri)

	mod, err := h.StateStore.Modules.ModuleByPath(dirHandle.Path())
	if err != nil {
		if state.IsModuleNotFound(err) {
			err = h.StateStore.Modules.Add(dirHandle.Path())
			if err != nil {
				return nil, err
			}
			mod, err = h.StateStore.Modules.ModuleByPath(dirHandle.Path())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	notifier, err := lsctx.DiagnosticsNotifier(ctx)
	if err != nil {
		return nil, err
	}

	progress.Begin(ctx, "Validating")

	validateDiags := make(map[string]hcl.Diagnostics, 0)
	for filename, f := range mod.ParsedModuleFiles {
		_, contentDiags := f.Body.Content(configFileSchema)
		validateDiags[string(filename)] = contentDiags
	}

	diags := diagnostics.NewDiagnostics()
	diags.EmptyRootDiagnostic()
	diags.Append("early validation", validateDiags)
	diags.Append("HCL", mod.ModuleDiagnostics.AutoloadedOnly().AsMap())
	diags.Append("HCL", mod.VarsDiagnostics.AutoloadedOnly().AsMap())

	notifier.PublishHCLDiags(ctx, mod.Path, diags)

	return nil, nil
}

// configFileSchema is the schema for the top-level of a config file. We use
// the low-level HCL API for this level so we can easily deal with each
// block type separately with its own decoding logic.
var configFileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "terraform",
		},
		{
			// This one is not really valid, but we include it here so we
			// can create a specialized error message hinting the user to
			// nest it inside a "terraform" block.
			Type: "required_providers",
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
		{
			Type: "locals",
		},
		{
			Type:       "output",
			LabelNames: []string{"name"},
		},
		{
			Type:       "module",
			LabelNames: []string{"name"},
		},
		{
			Type:       "resource",
			LabelNames: []string{"type", "name"},
		},
		{
			Type:       "data",
			LabelNames: []string{"type", "name"},
		},
		{
			Type: "moved",
		},
		{
			Type: "import",
		},
		{
			Type:       "check",
			LabelNames: []string{"name"},
		},
	},
}
