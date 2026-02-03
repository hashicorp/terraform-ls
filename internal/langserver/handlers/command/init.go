// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	"github.com/hashicorp/terraform-ls/internal/langserver/progress"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (h *CmdHandler) TerraformInitHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	dirUri, ok := args.GetString("uri")
	if !ok || dirUri == "" {
		return nil, fmt.Errorf("%w: expected module uri argument to be set", jrpc2.InvalidParams.Err())
	}

	if !uri.IsURIValid(dirUri) {
		return nil, fmt.Errorf("URI %q is not valid", dirUri)
	}

	dirHandle := document.DirHandleFromURI(dirUri)
	tfExec, err := module.TerraformExecutorForModule(ctx, dirHandle.Path())
	if err != nil {
		return nil, errors.EnrichTfExecError(err)
	}

	progress.Begin(ctx, "Initializing")
	defer func() {
		progress.End(ctx, "Finished")
	}()

	progress.Report(ctx, "Running terraform init ...")
	err = tfExec.Init(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
