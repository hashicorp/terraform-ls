// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	"github.com/hashicorp/terraform-ls/internal/langserver/progress"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (h *CmdHandler) TerraformValidateHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
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

	tfExec, err := module.TerraformExecutorForModule(ctx, mod.Path)
	if err != nil {
		return nil, errors.EnrichTfExecError(err)
	}

	notifier, err := lsctx.DiagnosticsNotifier(ctx)
	if err != nil {
		return nil, err
	}

	progress.Begin(ctx, "Validating")
	defer func() {
		progress.End(ctx, "Finished")
	}()
	progress.Report(ctx, "Running terraform validate ...")
	jsonDiags, err := tfExec.Validate(ctx)
	if err != nil {
		return nil, err
	}

	diags := diagnostics.NewDiagnostics()
	validateDiags := diagnostics.HCLDiagsFromJSON(jsonDiags)
	diags.EmptyRootDiagnostic()
	diags.Append("terraform validate", validateDiags)
	diags.Append("early validation", mod.ValidationDiagnostics)
	diags.Append("HCL", mod.ModuleDiagnostics.AutoloadedOnly().AsMap())
	diags.Append("HCL", mod.VarsDiagnostics.AutoloadedOnly().AsMap())

	notifier.PublishHCLDiags(ctx, mod.Path, diags)

	return nil, nil
}
