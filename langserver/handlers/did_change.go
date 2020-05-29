package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidChange(ctx context.Context, params DidChangeTextDocumentParams) error {
	p := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: params.TextDocument.URI,
			},
			Version: params.TextDocument.Version,
		},
		ContentChanges: params.ContentChanges,
	}

	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return err
	}

	fh := ilsp.VersionedFileHandler(p.TextDocument)
	f, err := fs.GetFile(fh)
	if err != nil {
		return err
	}

	if p.TextDocument.Version <= f.Version() {
		fs.Close(fh)
		return fmt.Errorf("Old version (%d) received, current version is %d. "+
			"Unable to update %s. This is likely a bug, please report it.",
			p.TextDocument.Version, f.Version(), p.TextDocument.URI)
	}

	if p.TextDocument.Version > f.Version()+1 {
		fs.Close(fh)
		return fmt.Errorf("New version (%d) received, current version is %d. "+
			"Unable to update %s. This is likely a bug, please report it.",
			p.TextDocument.Version, f.Version(), p.TextDocument.URI)
	}

	changes, err := ilsp.FileChanges(params.ContentChanges, f)
	if err != nil {
		return err
	}
	return fs.Change(fh, changes)
}

// TODO: Revisit after https://github.com/hashicorp/terraform-ls/issues/118 is addressed
// Then we could switch back to upstream go-lsp
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier      `json:"textDocument"`
	ContentChanges []lsp.TextDocumentContentChangeEvent `json:"contentChanges"`
}

type VersionedTextDocumentIdentifier struct {
	URI lsp.DocumentURI `json:"uri"`
	/**
	 * The version number of this document.
	 */
	Version int `json:"version"`
}

// UnmarshalJSON implements non-strict json.Unmarshaler.
func (v *DidChangeTextDocumentParams) UnmarshalJSON(b []byte) error {
	type t DidChangeTextDocumentParams
	return json.Unmarshal(b, (*t)(v))
}

// UnmarshalJSON implements non-strict json.Unmarshaler.
func (v *VersionedTextDocumentIdentifier) UnmarshalJSON(b []byte) error {
	type t VersionedTextDocumentIdentifier
	return json.Unmarshal(b, (*t)(v))
}
