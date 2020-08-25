package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type TokenizedFile interface {
	BlockAtPosition(hcl.Pos) (TokenizedBlock, error)
	TokenAtPosition(hcl.Pos) (hclsyntax.Token, error)
	PosInBlock(hcl.Pos) bool
	Blocks() ([]TokenizedBlock, error)
}

type TokenizedBlock interface {
	TokenAtPosition(hcl.Pos) (hclsyntax.Token, error)
	Tokens() hclsyntax.Tokens
}
