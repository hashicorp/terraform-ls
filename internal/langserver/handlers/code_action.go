package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (h *logHandler) TextDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) []lsp.CodeAction {
	ca, err := h.textDocumentCodeAction(ctx, params)
	if err != nil {
		h.logger.Printf("code action failed: %s", err)
	}

	return ca
}

func (h *logHandler) textDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	var ca []lsp.CodeAction

	var wantedCodeActions ilsp.CodeActions
	if len(params.Context.Only) == 0 {
		wantedCodeActions = ilsp.MinimalCodeActions
	} else {
		wantedCodeActions = ilsp.SupportedCodeActions.Only(params.Context.Only)
	}

	if len(wantedCodeActions) == 0 {
		return nil, fmt.Errorf("could not find a supported code action to execute for %s, wanted %v",
			params.TextDocument.URI, params.Context.Only)
	}

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return ca, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	file, err := fs.GetDocument(fh)
	if err != nil {
		return ca, err
	}

	for action := range wantedCodeActions {
		switch action {
		case lsp.QuickFix:
			pca, _ := prefillFieldsCodeAction(ctx, fh, file, params)
			ca = append(ca, pca...)
		case lsp.Source, lsp.SourceFixAll, ilsp.SourceFormatAll, ilsp.SourceFormatAllTerraformLs:
			original, err := file.Text()
			if err != nil {
				return ca, err
			}

			tfExec, err := module.TerraformExecutorForModule(ctx, fh.Dir())
			if err != nil {
				return ca, errors.EnrichTfExecError(err)
			}

			h.logger.Printf("formatting document via %q", tfExec.GetExecPath())

			edits, err := formatDocument(ctx, tfExec, original, file)
			if err != nil {
				return ca, err
			}

			ca = append(ca, lsp.CodeAction{
				Title: "Format Document",
				Kind:  lsp.SourceFixAll,
				Edit: lsp.WorkspaceEdit{
					Changes: map[string][]lsp.TextEdit{
						string(fh.URI()): edits,
					},
				},
			})
		}
	}

	return ca, nil
}

func prefillFieldsCodeAction(ctx context.Context, fh ilsp.FileHandler, file filesystem.Document, params lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	var ca []lsp.CodeAction

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return ca, err
	}

	mod, err := mf.ModuleByPath(file.Dir())
	if err != nil {
		return ca, err
	}

	schema, err := schemaForDocument(mf, file)
	if err != nil {
		return ca, err
	}

	d, err := decoderForDocument(ctx, mod, file.LanguageID())
	if err != nil {
		return ca, err
	}
	d.SetSchema(schema)
	d.PrefillRequiredFields = true
	p := lsp.Position{
		Line:      params.Range.Start.Line,
		Character: params.Range.Start.Character,
	}
	fPos, err := ilsp.FilePositionFromDocumentPosition2(p, params.TextDocument.URI, file)
	if err != nil {
		return ca, err
	}

	candidates, err := d.CandidatesAtPos(file.Filename(), fPos.Position())
	if err != nil {
		return ca, err
	}
	snippet := ""
	for _, c := range candidates.List {
		snippet += c.TextEdit.Snippet + "\n"
	}
	tes := []lsp.TextEdit{}
	tes = append(tes, lsp.TextEdit{
		Range: params.Range,
		NewText: snippet,
	})

	ca = append(ca, lsp.CodeAction{
		Title: "Fill in required fields",
		Kind:  lsp.QuickFix,
		Edit: lsp.WorkspaceEdit{
			Changes: map[string][]lsp.TextEdit{
				string(fh.URI()): tes,
			},
		},
	})
	return ca, nil
}

// p := lsp.Position{
// 	Line:      params.Range.Start.Line,
// 	Character: params.Range.Start.Character,
// }
// fPos, err := ilsp.FilePositionFromDocumentPosition2(p, params.TextDocument.URI, file)
// if err != nil {
// 	return ca, err
// }

// candidates, err := d.CandidatesAtPos(file.Filename(), fPos.Position())
// if err != nil {
// 	return ca, err
// }

// h.logger.Printf("received candidates: %#v", candidates)

// tes := []lsp.TextEdit{}
// snippet := ""
// // for i, c := range candidates.List {

// // 	h.logger.Printf("Processing %d: %s", i, c.Label)
// // 	snippet += "foo = \"${1:type}\"\n"
// // 	// t := lsp.TextEdit{
// // 		// 	Range: lsp.Range{
// // 			// 		Start: lsp.Position{
// // 				// 			Line:      uint32(c.TextEdit.Range.Start.Line),
// // 				// 			Character: uint32(c.TextEdit.Range.Start.Column),
// // 	// 		},
// // 	// 		End: lsp.Position{
// // 	// 			Line:      uint32(c.TextEdit.Range.End.Line),
// // 	// 			Character: uint32(c.TextEdit.Range.End.Column),
// // 	// 		},
// // 	// 	},
// // 	// 	NewText: c.TextEdit.NewText,
// // 	// }
// // 	// tes = append(tes, t)
// // }
// snippet += "foo = \"${1:type}\"\n"

// t := lsp.TextEdit{
// 	Range: lsp.Range{
// 		Start: lsp.Position{
// 			Line:      192,
// 			Character: 3,
// 		},
// 		End: lsp.Position{
// 			Line:      192,
// 			Character: 3,
// 		},
// 	},
// 	NewText: snippet,
// }
// tes = append(tes, t)

// ca = append(ca, lsp.CodeAction{
// 	Title: "Fill in required fields",
// 	Kind:  lsp.QuickFix,

// 	Edit: lsp.WorkspaceEdit{
// 		Changes: map[string][]lsp.TextEdit{
// 			string(fh.URI()): tes,
// 		},
// 		// DocumentChanges: []lsp.TextDocumentEdit{
// 		// 	{
// 		// 		Edits: tes,
// 		// 	},
// 		// },
// 	},
// })
// // ca = append(ca, lsp.CodeAction{
// // 	Title: "Fill in all fields",
// // 	Kind:  lsp.QuickFix,
// // 	Edit:  lsp.WorkspaceEdit{},
// // })
// // for _,r := range candidates.List {
// // 	ca = append(ca, lsp.CodeAction{
// // 		Title: "Fill in required fields",
// // 		Kind:  lsp.QuickFix,
// // 		Edit:  lsp.WorkspaceEdit{

// // 		},
// // 	})
// // }
