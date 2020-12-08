package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func TerraformValidateHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
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

	wasInit, _ := rm.WasInitialized()
	if !wasInit {
		return nil, fmt.Errorf("%s is not an initialized module, terraform validate cannot be called", dirUri)
	}

	diags, err := lsctx.Diagnostics(ctx)
	if err != nil {
		return nil, err
	}

	hclDiags, err := rm.ExecuteTerraformValidate(ctx)
	if err != nil {
		return nil, err
	}
	diags.PublishHCLDiags(ctx, rm.Path(), hclDiags, "terraform validate")

	return nil, nil
}
