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

	// lastEncodedTokenIdx tracks index of the last encoded token
	// so we can account for any skipped tokens in calculating diff
	lastEncodedTokenIdx int
}

func (te *TokenEncoder) Encode() []uint32 {
	data := make([]uint32, 0)

	for i := range te.Tokens {
		data = append(data, te.encodeTokenOfIndex(i)...)
	}

	return data
}

func (te *TokenEncoder) encodeTokenOfIndex(i int) []uint32 {
	token := te.Tokens[i]

	var tokenType TokenType
	modifiers := make([]TokenModifier, 0)

	switch token.Type {
	case lang.TokenBlockType:
		tokenType = TokenTypeType
	case lang.TokenBlockLabel:
		tokenType = TokenTypeEnumMember
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
	case lang.TokenKeyword:
		tokenType = TokenTypeVariable
	case lang.TokenTraversalStep:
		tokenType = TokenTypeVariable

	default:
		return []uint32{}
	}

	if !te.tokenTypeSupported(tokenType) {
		return []uint32{}
	}

	tokenTypeIdx := TokenTypesLegend(te.ClientCaps.TokenTypes).Index(tokenType)

	for _, m := range token.Modifiers {
		switch m {
		case lang.TokenModifierDependent:
			if !te.tokenModifierSupported(TokenModifierDefaultLibrary) {
				continue
			}
			modifiers = append(modifiers, TokenModifierDefaultLibrary)
		case lang.TokenModifierDeprecated:
			if !te.tokenModifierSupported(TokenModifierDeprecated) {
				continue
			}
			modifiers = append(modifiers, TokenModifierDeprecated)
		}
	}

	modifierBitMask := TokenModifiersLegend(te.ClientCaps.TokenModifiers).BitMask(modifiers)

	data := make([]uint32, 0)

	// Client may not support multiline tokens which would be indicated
	// via lsp.SemanticTokensCapabilities.MultilineTokenSupport
	// once it becomes available in gopls LSP structs.
	//
	// For now we just safely assume client does *not* support it.

	tokenLineDelta := token.Range.End.Line - token.Range.Start.Line

	previousLine := 0
	previousStartChar := 0
	if i > 0 {
		previousLine = te.Tokens[te.lastEncodedTokenIdx].Range.End.Line - 1
		currentLine := te.Tokens[i].Range.End.Line - 1
		if currentLine == previousLine {
			previousStartChar = te.Tokens[te.lastEncodedTokenIdx].Range.Start.Column - 1
		}
	}

	if tokenLineDelta == 0 || false /* te.clientCaps.MultilineTokenSupport */ {
		deltaLine := token.Range.Start.Line - 1 - previousLine
		tokenLength := token.Range.End.Byte - token.Range.Start.Byte
		deltaStartChar := token.Range.Start.Column - 1 - previousStartChar

		data = append(data, []uint32{
			uint32(deltaLine),
			uint32(deltaStartChar),
			uint32(tokenLength),
			uint32(tokenTypeIdx),
			uint32(modifierBitMask),
		}...)
	} else {
		// Add entry for each line of a multiline token
		for tokenLine := token.Range.Start.Line - 1; tokenLine <= token.Range.End.Line-1; tokenLine++ {
			deltaLine := tokenLine - previousLine

			deltaStartChar := 0
			if tokenLine == token.Range.Start.Line-1 {
				deltaStartChar = token.Range.Start.Column - 1 - previousStartChar
			}

			lineBytes := bytes.TrimRight(te.Lines[tokenLine].Bytes, "\n\r")
			length := len(lineBytes)

			if tokenLine == token.Range.End.Line-1 {
				length = token.Range.End.Column - 1
			}

			data = append(data, []uint32{
				uint32(deltaLine),
				uint32(deltaStartChar),
				uint32(length),
				uint32(tokenTypeIdx),
				uint32(modifierBitMask),
			}...)

			previousLine = tokenLine
		}
	}

	te.lastEncodedTokenIdx = i

	return data
}

func (te *TokenEncoder) tokenTypeSupported(tokenType TokenType) bool {
	return sliceContains(te.ClientCaps.TokenTypes, string(tokenType))
}

func (te *TokenEncoder) tokenModifierSupported(tokenModifier TokenModifier) bool {
	return sliceContains(te.ClientCaps.TokenModifiers, string(tokenModifier))
}
