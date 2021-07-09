package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentCodeLens(ctx context.Context, params lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	list := make([]lsp.CodeLens, 0)

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return list, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	file, err := fs.GetDocument(fh)
	if err != nil {
		return list, err
	}

	list = append(list, referenceCountCodeLens(ctx, file)...)

	return list, nil
}

func referenceCountCodeLens(ctx context.Context, doc filesystem.Document) []lsp.CodeLens {
	list := make([]lsp.CodeLens, 0)

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return list
	}

	showReferencesCmd, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).ShowReferencesCommandId()
	if !ok {
		return list
	}

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return list
	}

	mod, err := mf.ModuleByPath(doc.Dir())
	if err != nil {
		return list
	}

	schema, err := schemaForDocument(mf, doc)
	if err != nil {
		return list
	}

	d, err := decoderForDocument(ctx, mod, doc.LanguageID())
	if err != nil {
		return list
	}
	d.SetSchema(schema)

	refTargets, err := d.ReferenceTargetsInFile(doc.Filename())
	if err != nil {
		return list
	}

	refContext := lsp.ReferenceContext{}
	refContextBytes, err := json.Marshal(refContext)
	if err != nil {
		return list
	}

	for _, refTarget := range refTargets {
		origins, err := d.ReferenceOriginsTargeting(refTarget)
		if err != nil {
			continue
		}
		if len(origins) == 0 {
			continue
		}

		var pos hcl.Pos
		if refTarget.DefRangePtr != nil {
			pos = posMiddleOfRange(refTarget.DefRangePtr)
		} else {
			pos = posMiddleOfRange(refTarget.RangePtr)
		}

		posBytes, err := json.Marshal(ilsp.HCLPosToLSP(pos))
		if err != nil {
			return list
		}

		list = append(list, lsp.CodeLens{
			Range: ilsp.HCLRangeToLSP(*refTarget.RangePtr),
			Command: lsp.Command{
				Title:   getTitle("reference", "references", len(origins)),
				Command: showReferencesCmd,
				Arguments: []json.RawMessage{
					json.RawMessage(posBytes),
					json.RawMessage(refContextBytes),
				},
			},
		})
	}
	return list
}

func posMiddleOfRange(rng *hcl.Range) hcl.Pos {
	col := rng.Start.Column
	byte := rng.Start.Byte

	if rng.Start.Line == rng.End.Line && rng.End.Column > rng.Start.Column {
		charsFromStart := (rng.End.Column - rng.Start.Column) / 2
		col += charsFromStart
		byte += charsFromStart
	}

	return hcl.Pos{
		Line:   rng.Start.Line,
		Column: col,
		Byte:   byte,
	}
}

func getTitle(singular, plural string, n int) string {
	if n > 1 || n == 0 {
		return fmt.Sprintf("%d %s", n, plural)
	}
	return fmt.Sprintf("%d %s", n, singular)
}
