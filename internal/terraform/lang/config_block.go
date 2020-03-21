package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	lsp "github.com/sourcegraph/go-lsp"
)

type ConfigBlock interface {
	CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error)
	Name() string
	BlockType() string
}

type configBlockFactory interface {
	New(*hclsyntax.Block) (ConfigBlock, error)
}
