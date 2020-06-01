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

func wordBeforePos(tokens hclsyntax.Tokens, pos hcl.Pos) string {
	switch token := tokenAtPos(tokens, pos); token.Type {
	case hclsyntax.TokenIdent, hclsyntax.TokenQuotedLit, hclsyntax.TokenStringLit:
		return string(token.Bytes[:pos.Byte-token.Range.Start.Byte])
	default:
		return ""
	}
}

func tokenAtPos(tokens hclsyntax.Tokens, pos hcl.Pos) hclsyntax.Token {
	for _, t := range tokens {
		if rangeContainsOffset(t.Range, pos.Byte) {
			return t
		}
	}
	return hclsyntax.Token{
		Type: hclsyntax.TokenNil,
	}
}

// rangeContainsOffset is a reimplementation of hcl.Range.ContainsOffset
// which treats offset matching the end of a range as contained
func rangeContainsOffset(rng hcl.Range, offset int) bool {
	return offset >= rng.Start.Byte && offset <= rng.End.Byte
}
