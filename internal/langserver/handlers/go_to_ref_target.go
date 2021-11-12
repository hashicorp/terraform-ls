package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/lang"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) GoToReferenceTarget(ctx context.Context, params lsp.TextDocumentPositionParams) (interface{}, error) {
	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}

	fPos, err := ilsp.FilePositionFromDocumentPosition(params, doc)
	if err != nil {
		return nil, err
	}

	path := lang.Path{
		Path:       doc.Dir(),
		LanguageID: doc.LanguageID(),
	}
	targets, err := svc.decoder.ReferenceTargetsForOriginAtPos(path, doc.Filename(), fPos.Position())
	if err != nil {
		return nil, err
	}

	return ilsp.RefTargetsToLocationLinks(targets, cc.TextDocument.Declaration.LinkSupport), nil
}
