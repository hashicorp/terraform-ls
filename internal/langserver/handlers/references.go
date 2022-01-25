package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/lang"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) References(ctx context.Context, params lsp.ReferenceParams) ([]lsp.Location, error) {
	list := make([]lsp.Location, 0)

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return list, err
	}

	pos, err := ilsp.HCLPositionFromLspPosition(params.TextDocumentPositionParams.Position, doc)
	if err != nil {
		return list, err
	}

	path := lang.Path{
		Path:       doc.Dir.Path(),
		LanguageID: doc.LanguageID,
	}

	origins := svc.decoder.ReferenceOriginsTargetingPos(path, doc.Filename, pos)

	return ilsp.RefOriginsToLocations(origins), nil
}
