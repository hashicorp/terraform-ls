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

func TerraformPlanHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
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

	wasInit, err := rm.WasInitialized()
	if err != nil {
		return nil, fmt.Errorf("error checking if %s was initialized: %s", dirUri, err)
	}
	if !wasInit {
		return nil, fmt.Errorf("%s is not an initialized module, terraform validate cannot be called", dirUri)
	}

	progress.Begin(ctx, "Planning")
	defer func() {
		progress.End(ctx, "Finished")
	}()

	progress.Report(ctx, "Running terraform plan ...")
	err = rm.ExecuteTerraformPlan(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
