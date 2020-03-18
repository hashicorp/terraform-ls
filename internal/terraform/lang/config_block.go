package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

type ConfigBlock interface {
	CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error)
	Name() string
	BlockType() string
}

type configBlockFactory interface {
	New(*hcl.Block) (ConfigBlock, error)
}
