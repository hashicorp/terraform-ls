package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/lang"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) TextDocumentCodeLens(ctx context.Context, params lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	list := make([]lsp.CodeLens, 0)

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return list, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	doc, err := fs.GetDocument(fh)
	if err != nil {
		return list, err
	}

	path := lang.Path{
		Path:       doc.Dir(),
		LanguageID: doc.LanguageID(),
	}

	lenses, err := svc.decoder.CodeLensesForFile(ctx, path, doc.Filename())
	if err != nil {
		return nil, err
	}

	for _, lens := range lenses {
		cmd, err := ilsp.Command(lens.Command)
		if err != nil {
			svc.logger.Printf("skipping code lens %#v: %s", lens.Command, err)
			continue
		}

		list = append(list, lsp.CodeLens{
			Range:   ilsp.HCLRangeToLSP(lens.Range),
			Command: cmd,
		})
	}

	return list, nil
}
