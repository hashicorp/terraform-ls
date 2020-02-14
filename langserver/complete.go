package langserver

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/radeksimko/terraform-ls/internal/filesystem"
	"github.com/radeksimko/terraform-ls/internal/terraform"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentComplete(ctx context.Context, params lsp.CompletionParams) (lsp.CompletionList, error) {
	var list lsp.CompletionList

	fs := ctx.Value(ctxFs).(filesystem.Filesystem)

	log.Printf("Finding block at position %#v", params.TextDocumentPositionParams)
	hclBlock, hclPos, err := fs.HclBlockAtDocPosition(params.TextDocumentPositionParams)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}
	log.Printf("HCL block found at HCL pos %#v", hclPos)

	cfgBlock, err := terraform.ConfigBlockFromHcl(hclBlock)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}

	uri := filesystem.URI(params.TextDocumentPositionParams.TextDocument.URI)
	wd := uri.Dir()

	log.Printf("Retrieving schemas for %q ...", wd)
	start := time.Now()
	tf := terraform.TerraformExec(wd)
	schemas, err := tf.ProviderSchemas()
	if err != nil {
		return list, fmt.Errorf("unable to get schemas: %s", err)
	}
	log.Printf("Schemas retrieved in %s ...", time.Since(start))

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
