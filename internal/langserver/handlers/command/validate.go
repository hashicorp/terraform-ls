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

func TerraformValidateHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	dirUri, ok := args.GetString("uri")
	if !ok || dirUri == "" {
		return nil, fmt.Errorf("%w: expected dir uri argument to be set", code.InvalidParams.Err())
	}

	dh := ilsp.FileHandlerFromDirURI(lsp.DocumentURI(dirUri))

	cf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return nil, err
	}

	mod, err := cf.ModuleByPath(dh.Dir())
	if err != nil {
		return nil, err
	}

	wasInit, err := mod.WasInitialized()
	if err != nil {
		return nil, fmt.Errorf("error checking if %s was initialized: %s", dirUri, err)
	}
	if !wasInit {
		return nil, fmt.Errorf("%s is not an initialized module, terraform validate cannot be called", dirUri)
	}

	diags, err := lsctx.Diagnostics(ctx)
	if err != nil {
		return nil, err
	}

	progress.Begin(ctx, "Validating")
	defer func() {
		progress.End(ctx, "Finished")
	}()
	progress.Report(ctx, "Running terraform validate ...")
	hclDiags, err := mod.ExecuteTerraformValidate(ctx)
	if err != nil {
		return nil, err
	}
	diags.PublishHCLDiags(ctx, mod.Path(), hclDiags, "terraform validate")

	return nil, nil
}
