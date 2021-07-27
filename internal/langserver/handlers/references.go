package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/lang"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) References(ctx context.Context, params lsp.ReferenceParams) ([]lsp.Location, error) {
	list := make([]lsp.Location, 0)

	fs, err := lsctx.DocumentStorage(ctx)
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

	schema, err := schemaForDocument(mf, file)
	if err != nil {
		return list, err
	}

	d, err := decoderForDocument(ctx, mod, file.LanguageID())
	if err != nil {
		return list, err
	}
	d.SetSchema(schema)

	fPos, err := ilsp.FilePositionFromDocumentPosition(params.TextDocumentPositionParams, file)
	if err != nil {
		return list, err
	}

	refTargets, err := d.InnermostReferenceTargetsAtPos(fPos.Filename(), fPos.Position())
	if err != nil {
		return list, err
	}
	if len(refTargets) == 0 {
		// this position is not addressable
		h.logger.Printf("position is not addressable: %s - %#v", fPos.Filename(), fPos.Position())
		return list, nil
	}

	h.logger.Printf("finding origins for inner-most targets: %#v", refTargets)

	origins := make(lang.ReferenceOrigins, 0)
	for _, refTarget := range refTargets {
		refOrigins, err := d.ReferenceOriginsTargeting(refTarget)
		if err != nil {
			return list, err
		}
		origins = append(origins, refOrigins...)
	}

	return ilsp.RefOriginsToLocations(mod.Path, origins), nil
}
