package handlers

import (
	"context"
	"fmt"
	"time"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	lsp "github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentComplete(ctx context.Context, params lsp.CompletionParams) (lsp.CompletionList, error) {
	var list lsp.CompletionList

	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return list, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return list, err
	}

	h.logger.Printf("Finding block at position %#v", params.TextDocumentPositionParams)
	hclBlock, hclPos, err := fs.HclBlockAtDocPosition(params.TextDocumentPositionParams)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}
	h.logger.Printf("HCL block found at HCL pos %#v", hclPos)

	p := lang.NewParserWithLogger(h.logger)
	p.SetCapabilities(cc.TextDocument)

	cfgBlock, err := p.ParseBlockFromHcl(hclBlock)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}

	uri := fs.URI(params.TextDocumentPositionParams.TextDocument.URI)
	wd := uri.Dir()

	h.logger.Printf("Retrieving schemas for %q ...", wd)

	start := time.Now()

	tf, err := lsctx.TerraformExecutor(ctx)
	if err != nil {
		return list, err
	}
	tf.SetWorkdir(wd)

	schemas, err := tf.ProviderSchemas()
	if err != nil {
		return list, fmt.Errorf("unable to get schemas: %s", err)
	}
	h.logger.Printf("Schemas retrieved in %s ...", time.Since(start))

	err = cfgBlock.LoadSchema(schemas)
	if err != nil {
		return list, fmt.Errorf("loading schema failed: %s", err)
	}

	list, err = cfgBlock.CompletionItemsAtPos(hclPos)
	if err != nil {
		return list, fmt.Errorf("finding completion items failed: %s", err)
	}

	return list, nil
}
