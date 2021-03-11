package lsp

import (
	"bytes"

	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/source"
)

type TokenEncoder struct {
	Lines      source.Lines
	Tokens     []lang.SemanticToken
	ClientCaps lsp.SemanticTokensClientCapabilities
}

func (te *TokenEncoder) Encode() []float64 {
	data := make([]float64, 0)

	for i := range te.Tokens {
		data = append(data, te.encodeTokenOfIndex(i)...)
	}

	return data
}

func (te *TokenEncoder) encodeTokenOfIndex(i int) []float64 {
	token := te.Tokens[i]

	var tokenType TokenType
	modifiers := make([]TokenModifier, 0)

	switch token.Type {
	case lang.TokenBlockType:
		tokenType = TokenTypeType
	case lang.TokenBlockLabel:
		tokenType = TokenTypeString
	case lang.TokenAttrName:
		tokenType = TokenTypeProperty
	case lang.TokenBool:
		tokenType = TokenTypeKeyword
	case lang.TokenNumber:
		tokenType = TokenTypeNumber
	case lang.TokenString:
		tokenType = TokenTypeString
	case lang.TokenObjectKey:
		tokenType = TokenTypeParameter
	case lang.TokenMapKey:
		tokenType = TokenTypeParameter

	default:
		return []float64{}
	}

	if !te.tokenTypeSupported(tokenType) {
		return []float64{}
	}

	tokenTypeIdx := TokenTypesLegend(te.ClientCaps.TokenTypes).Index(tokenType)

	for _, m := range token.Modifiers {
		switch m {
		case lang.TokenModifierDependent:
			if !te.tokenModifierSupported(TokenModifierModification) {
				continue
			}
			modifiers = append(modifiers, TokenModifierModification)
		case lang.TokenModifierDeprecated:
			if !te.tokenModifierSupported(TokenModifierDeprecated) {
				continue
			}
			modifiers = append(modifiers, TokenModifierDeprecated)
		}
	}

	modifierBitMask := TokenModifiersLegend(te.ClientCaps.TokenModifiers).BitMask(modifiers)

	data := make([]float64, 0)

	// Client may not support multiline tokens which would be indicated
	// via lsp.SemanticTokensCapabilities.MultilineTokenSupport
	// once it becomes available in gopls LSP structs.
	//
	// For now we just safely assume client does *not* support it.

	tokenLineDelta := token.Range.End.Line - token.Range.Start.Line

	previousLine := 0
	previousStartChar := 0
	if i > 0 {
		previousLine = te.Tokens[i-1].Range.End.Line - 1
		currentLine := te.Tokens[i].Range.End.Line - 1
		if currentLine == previousLine {
			previousStartChar = te.Tokens[i-1].Range.Start.Column - 1
		}
	}

	if tokenLineDelta == 0 || false /* te.clientCaps.MultilineTokenSupport */ {
		deltaLine := token.Range.Start.Line - 1 - previousLine
		tokenLength := token.Range.End.Byte - token.Range.Start.Byte
		deltaStartChar := token.Range.Start.Column - 1 - previousStartChar

		data = append(data, []float64{
			float64(deltaLine),
			float64(deltaStartChar),
			float64(tokenLength),
			float64(tokenTypeIdx),
			float64(modifierBitMask),
		}...)
	} else {
		// Add entry for each line of a multiline token
		for tokenLine := token.Range.Start.Line - 1; tokenLine <= token.Range.End.Line-1; tokenLine++ {
			deltaLine := tokenLine - previousLine

			deltaStartChar := 0
			if tokenLine == token.Range.Start.Line-1 {
				deltaStartChar = token.Range.Start.Column - 1 - previousStartChar
			}

			lineBytes := bytes.TrimRight(te.Lines[tokenLine].Bytes(), "\n\r")
			length := len(lineBytes)

			if tokenLine == token.Range.End.Line-1 {
				length = token.Range.End.Column - 1
			}

			data = append(data, []float64{
				float64(deltaLine),
				float64(deltaStartChar),
				float64(length),
				float64(tokenTypeIdx),
				float64(modifierBitMask),
			}...)

			previousLine = tokenLine
		}
	}

	return data
}

func (te *TokenEncoder) tokenTypeSupported(tokenType TokenType) bool {
	return sliceContains(te.ClientCaps.TokenTypes, string(tokenType))
}

func (te *TokenEncoder) tokenModifierSupported(tokenModifier TokenModifier) bool {
	return sliceContains(te.ClientCaps.TokenModifiers, string(tokenModifier))
}
