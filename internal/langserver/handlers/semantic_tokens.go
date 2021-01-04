package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (lh *logHandler) TextDocumentSemanticTokensFull(ctx context.Context, params lsp.SemanticTokensParams) (lsp.SemanticTokens, error) {
	tks := lsp.SemanticTokens{}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return tks, err
	}

	caps := ilsp.SemanticTokensClientCapabilities{
		SemanticTokensClientCapabilities: cc.TextDocument.SemanticTokens,
	}
	if !caps.FullRequest() {
		// This would indicate a buggy client which sent a request
		// it didn't claim to support, so we just strictly follow
		// the protocol here and avoid serving buggy clients.
		lh.logger.Printf("semantic tokens full request support not announced by client")
		return tks, code.MethodNotFound.Err()
	}

	ds, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return tks, err
	}

	rmf, err := lsctx.RootModuleFinder(ctx)
	if err != nil {
		return tks, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	doc, err := ds.GetDocument(fh)
	if err != nil {
		return tks, err
	}

	rm, err := rmf.RootModuleByPath(doc.Dir())
	if err != nil {
		return tks, fmt.Errorf("finding compatible decoder failed: %w", err)
	}

	schema, err := rmf.SchemaForPath(doc.Dir())
	if err != nil {
		return tks, err
	}

	d, err := rm.DecoderWithSchema(schema)
	if err != nil {
		return tks, err
	}

	tokens, err := d.SemanticTokensInFile(doc.Filename())
	if err != nil {
		return tks, err
	}

	te := &ilsp.TokenEncoder{
		Lines:      doc.Lines(),
		Tokens:     tokens,
		ClientCaps: cc.TextDocument.SemanticTokens,
	}
	tks.Data = te.Encode()

	return tks, nil
}
