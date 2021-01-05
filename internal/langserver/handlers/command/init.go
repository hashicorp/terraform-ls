package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/progress"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func TerraformInitHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	dirUri, ok := args.GetString("uri")
	if !ok || dirUri == "" {
		return nil, fmt.Errorf("%w: expected dir uri argument to be set", code.InvalidParams.Err())
	}

	dh := ilsp.FileHandlerFromDirURI(lsp.DocumentURI(dirUri))

	cf, err := lsctx.RootModuleFinder(ctx)
	if err != nil {
		return nil, err
	}

	rm, err := cf.RootModuleByPath(dh.Dir())
	if err != nil {
		return nil, err
	}

	progress.Begin(ctx, "Initializing")
	defer func() {
		progress.End(ctx, "Finished")
	}()

	progress.Report(ctx, "Running terraform init ...")
	err = rm.ExecuteTerraformInit(ctx)
	if err != nil {
		return nil, err
	}

	progress.Report(ctx, "Detecting paths to watch ...")
	paths := rm.PathsToWatch()

	w, err := lsctx.Watcher(ctx)
	if err != nil {
		return nil, err
	}
	err = w.AddPaths(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to add watch for dir (%s): %+v", dh.Dir(), err)
	}

	return nil, nil
}
