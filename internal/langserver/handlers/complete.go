package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/decoder"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentComplete(ctx context.Context, params lsp.CompletionParams) (lsp.CompletionList, error) {
	var list lsp.CompletionList

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return list, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return list, err
	}

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return list, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return list, err
	}

	mod, err := mf.ModuleByPath(file.Dir())
	if err != nil {
		return list, err
	}

	schema, err := mf.SchemaForModule(file.Dir())
	if err != nil {
		return list, err
	}

	d, err := decoder.DecoderForModule(ctx, mod)
	if err != nil {
		return list, err
	}
	d.SetSchema(schema)

	fPos, err := ilsp.FilePositionFromDocumentPosition(params.TextDocumentPositionParams, file)
	if err != nil {
		return list, err
	}

	h.logger.Printf("Looking for candidates at %q -> %#v", file.Filename(), fPos.Position())
	candidates, err := d.CandidatesAtPos(file.Filename(), fPos.Position())
	h.logger.Printf("received candidates: %#v", candidates)
	return ilsp.ToCompletionList(candidates, cc.TextDocument), err
}
