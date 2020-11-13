package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	"github.com/sourcegraph/go-lsp"
)

func HoverData(data *lang.HoverData, cc lsp.TextDocumentClientCapabilities) lsp.Hover {
	mdSupported := cc.Hover != nil &&
		len(cc.Hover.ContentFormat) > 0 &&
		cc.Hover.ContentFormat[0] == "markdown"

	value := data.Content.Value
	if data.Content.Kind == lang.MarkdownKind && !mdSupported {
		value = mdplain.Clean(value)
	}

	content := lsp.RawMarkedString(value)
	rng := HCLRangeToLSP(data.Range)

	return lsp.Hover{
		Contents: []lsp.MarkedString{content},
		Range:    &rng,
	}
}
