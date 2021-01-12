package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (h *logHandler) TextDocumentFormatting(ctx context.Context, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return edits, err
	}

	tff, err := lsctx.TerraformFormatterFinder(ctx)
	if err != nil {
		return edits, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	file, err := fs.GetDocument(fh)
	if err != nil {
		return edits, err
	}

	format, err := findTerraformFormatter(ctx, tff, file.Dir())
	if err != nil {
		return edits, err
	}

	original, err := file.Text()
	if err != nil {
		return edits, err
	}

	formatted, err := format(ctx, original)
	if err != nil {
		return edits, err
	}

	changes := hcl.Diff(file, original, formatted)

	return ilsp.TextEditsFromDocumentChanges(changes), nil
}

func findTerraformFormatter(ctx context.Context, tff module.TerraformFormatterFinder, dir string) (exec.Formatter, error) {
	discoveryDone, err := tff.HasTerraformDiscoveryFinished(dir)
	if err != nil {
		if module.IsModuleNotFound(err) {
			return tff.TerraformFormatterForDir(ctx, dir)
		}
		return nil, err
	} else {
		if !discoveryDone {
			// TODO: block until it's available <-tff.TerraformLoadingDone()
			return nil, fmt.Errorf("terraform is still being discovered for %s", dir)
		}
		available, err := tff.IsTerraformAvailable(dir)
		if err != nil {
			if module.IsModuleNotFound(err) {
				return tff.TerraformFormatterForDir(ctx, dir)
			}
		}
		if !available {
			// TODO: block until it's available <-tff.TerraformLoadingDone()
			return nil, fmt.Errorf("terraform is not available for %s", dir)
		}
	}

	return tff.TerraformFormatterForDir(ctx, dir)
}
