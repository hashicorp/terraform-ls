package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	"github.com/hashicorp/terraform-ls/internal/langserver/progress"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TerraformInitHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	dirUri, ok := args.GetString("uri")
	if !ok || dirUri == "" {
		return nil, fmt.Errorf("%w: expected module uri argument to be set", code.InvalidParams.Err())
	}

	if !uri.IsURIValid(dirUri) {
		return nil, fmt.Errorf("URI %q is not valid", dirUri)
	}

	dh := ilsp.FileHandlerFromDirURI(lsp.DocumentURI(dirUri))

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return nil, err
	}

	mod, err := modMgr.ModuleByPath(dh.Dir())
	if err != nil {
		if module.IsModuleNotFound(err) {
			mod, err = modMgr.AddModule(dh.Dir())
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
