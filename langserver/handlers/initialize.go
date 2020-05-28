package handlers

import (
	"context"
	"errors"
	"fmt"
	"os"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	tferr "github.com/hashicorp/terraform-ls/internal/terraform/errors"
	lsp "github.com/sourcegraph/go-lsp"
)

func (lh *logHandler) Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Options: &lsp.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    lsp.TDSKIncremental,
				},
			},
			CompletionProvider: &lsp.CompletionOptions{
				ResolveProvider: false,
			},
			DocumentFormattingProvider: true,
		},
	}

	fh := ilsp.FileHandler(params.RootURI)
	if !fh.Valid() {
		return serverCaps, fmt.Errorf("URI %q is not valid", params.RootURI)
	}

	rootURI := fh.FullPath()

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

	// Workdir is not important for version
	// and schema obtaining is explicit anyway
	tf.SetWorkdir(os.TempDir())

	tfVersion, err := tf.Version()
	if err != nil {
		return serverCaps, err
	}

	err = supportsTerraform(tfVersion)
	if err != nil {
		if uvErr, ok := err.(*tferr.UnsupportedTerraformVersion); ok {
			lh.logger.Printf("Unsupported terraform version: %s", uvErr)
			// Which component exactly imposed the constrain may not be relevant
			// to the user unless they are very familiar with internals of the LS
			// so we avoid displaying it, but it will be logged for debugging purposes.
			uvErr.Component = ""

			return serverCaps, fmt.Errorf("%w. "+
				"Please upgrade or make supported version available in $PATH"+
				" and reopen %s", uvErr, rootURI)
		}

		// We naively assume that Terraform version can't change at runtime
		// and just fail initalization early and force user to reopen IDE
		// with supported TF version.
		//
		// Longer-term we may want to pick up changes while LS is running.
		// That would require asynchronous and continuous discovery though.
		return serverCaps, err
	}

	lh.logger.Printf("Found compatible Terraform version (%s) at %s",
		tfVersion, tf.GetExecPath())

	err = lsctx.SetTerraformVersion(ctx, tfVersion)
	if err != nil {
		return serverCaps, err
	}

	err = ss.ObtainSchemasForWorkspace(tf, rootURI)
	if err != nil {
		return serverCaps, err
	}

	err = ss.AddWorkspaceForWatching(rootURI)
	if err != nil {
		if errors.Is(err, &tferr.NotInitializedErr{}) {
			return serverCaps, fmt.Errorf("Workspace not initialized. "+
				"Please run `terraform init` in %s", rootURI)
		}
		return serverCaps, err
	}

	err = ss.StartWatching(tf)
	if err != nil {
		return serverCaps, err
	}

	return serverCaps, nil
}
