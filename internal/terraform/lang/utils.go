package lang

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func PosInLabels(b *hclsyntax.Block, pos hcl.Pos) bool {
	if b == nil {
		return false
	}

	for _, rng := range b.LabelRanges {
		if rangeContainsOffset(rng, pos.Byte) {
			return true
		}
	}
	return false
}

func prefixAtPos(looker TokenAtPosLooker, pos hcl.Pos) string {
	token, err := looker.TokenAtPosition(pos)
	if err != nil {
		return ""
	}

	switch token.Type {
	case hclsyntax.TokenIdent, hclsyntax.TokenQuotedLit, hclsyntax.TokenStringLit:
		return string(token.Bytes[:pos.Byte-token.Range.Start.Byte])
	}

	return ""
}

type TokenAtPosLooker interface {
	TokenAtPosition(hcl.Pos) (hclsyntax.Token, error)
}

// rangeContainsOffset is a reimplementation of hcl.Range.ContainsOffset
// which treats offset matching the end of a range as contained
func rangeContainsOffset(rng hcl.Range, offset int) bool {
	return offset >= rng.Start.Byte && offset <= rng.End.Byte
}
