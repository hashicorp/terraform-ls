package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	fs "github.com/hashicorp/terraform-ls/internal/filesystem"
	lsp "github.com/sourcegraph/go-lsp"
)

func Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Options: &lsp.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    lsp.TDSKFull,
				},
			},
			CompletionProvider: &lsp.CompletionOptions{
				ResolveProvider: false,
			},
		},
	}

	uri := fs.URI(params.RootURI)
	if !uri.Valid() {
		return serverCaps, fmt.Errorf("URI %q is not valid", params.RootURI)
	}

	rootURI := uri.FullPath()

	if rootURI == "" {
		return serverCaps, fmt.Errorf("Editing a single file is not yet supported." +
			" Please open a directory.")
	}

	err := lsctx.SetClientCapabilities(ctx, &params.Capabilities)
	if err != nil {
		return serverCaps, err
	}

	ss, err := lsctx.TerraformSchemaWriter(ctx)
	if err != nil {
		return serverCaps, err
	}

	tf, err := lsctx.TerraformExecutor(ctx)
	if err != nil {
		return serverCaps, err
	}

	tf.SetWorkdir(rootURI)

	tfVersion, err := tf.Version()
	if err != nil {
		return serverCaps, err
	}

	err = lsctx.SetTerraformVersion(ctx, tfVersion)
	if err != nil {
		return serverCaps, err
	}

	err = ss.ObtainSchemasForDir(tf, rootURI)
	if err != nil {
		return serverCaps, err
	}

	return serverCaps, nil
}
