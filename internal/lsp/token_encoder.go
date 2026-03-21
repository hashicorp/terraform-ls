// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"bytes"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/lsp/semtok"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/source"
)

type TokenEncoder struct {
	Lines      source.Lines
	Tokens     []lang.SemanticToken
	ClientCaps lsp.SemanticTokensClientCapabilities

	lastEncodedLine      int
	lastEncodedStartChar int
	hasEncodedToken      bool
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

	tokenType, ok := te.resolveTokenType(token)
	if !ok {
		return []uint32{}
	}
	tokenTypeIdx := TokenTypesLegend(te.ClientCaps.TokenTypes).Index(tokenType)

	modifiers := te.resolveTokenModifiers(token.Modifiers)
	modifierBitMask := TokenModifiersLegend(te.ClientCaps.TokenModifiers).BitMask(modifiers)

	data := make([]uint32, 0)

	// Client may not support multiline tokens which would be indicated
	// via lsp.SemanticTokensCapabilities.MultilineTokenSupport
	// once it becomes available in gopls LSP structs.
	//
	// For now we just safely assume client does *not* support it.

	tokenLineDelta := token.Range.End.Line - token.Range.Start.Line

	if tokenLineDelta == 0 || false /* te.clientCaps.MultilineTokenSupport */ {
		previousLine, previousStartChar := te.lastPosition(token.Range.Start.Line - 1)
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

		te.recordPosition(token.Range.Start.Line-1, token.Range.Start.Column-1)
	} else {
		// Add entry for each line of a multiline token
		for tokenLine := token.Range.Start.Line - 1; tokenLine <= token.Range.End.Line-1; tokenLine++ {
			startChar := 0
			lineBytes := bytes.TrimRight(te.Lines[tokenLine].Bytes, "\n\r")
			length := len(lineBytes)
			if tokenLine == token.Range.Start.Line-1 {
				startChar = token.Range.Start.Column - 1
				length -= startChar
			}

			if tokenLine == token.Range.End.Line-1 {
				length = token.Range.End.Column - 1 - startChar
			}

			if length <= 0 {
				continue
			}

			previousLine, previousStartChar := te.lastPosition(tokenLine)
			deltaLine := tokenLine - previousLine
			deltaStartChar := startChar - previousStartChar

			data = append(data, []uint32{
				uint32(deltaLine),
				uint32(deltaStartChar),
				uint32(length),
				uint32(tokenTypeIdx),
				uint32(modifierBitMask),
			}...)

			te.recordPosition(tokenLine, startChar)
		}
	}

	return data
}

func (te *TokenEncoder) lastPosition(tokenLine int) (int, int) {
	if !te.hasEncodedToken {
		return 0, 0
	}

	if tokenLine == te.lastEncodedLine {
		return te.lastEncodedLine, te.lastEncodedStartChar
	}

	return te.lastEncodedLine, 0
}

func (te *TokenEncoder) recordPosition(line, startChar int) {
	te.lastEncodedLine = line
	te.lastEncodedStartChar = startChar
	te.hasEncodedToken = true
}

func (te *TokenEncoder) resolveTokenType(token lang.SemanticToken) (semtok.TokenType, bool) {
	switch token.Type {
	case lang.TokenBlockType:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenBlockType), semtok.TokenTypeType)
	case lang.TokenBlockLabel:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenBlockLabel), semtok.TokenTypeEnumMember)
	case lang.TokenAttrName:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenAttrName), semtok.TokenTypeProperty)
	case lang.TokenBool:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenBool), semtok.TokenTypeKeyword)
	case lang.TokenNumber:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenNumber), semtok.TokenTypeNumber)
	case lang.TokenString:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenString), semtok.TokenTypeString)
	case lang.TokenObjectKey:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenObjectKey), semtok.TokenTypeParameter)
	case lang.TokenMapKey:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenMapKey), semtok.TokenTypeParameter)
	case lang.TokenKeyword:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenKeyword), semtok.TokenTypeVariable)
	case lang.TokenReferenceStep:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenReferenceStep), semtok.TokenTypeVariable)
	case lang.TokenTypeComplex:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenTypeComplex), semtok.TokenTypeFunction)
	case lang.TokenTypePrimitive:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenTypePrimitive), semtok.TokenTypeKeyword)
	case lang.TokenFunctionName:
		return te.firstSupportedTokenType(
			semtok.TokenType(lang.TokenFunctionName), semtok.TokenTypeFunction)
	}

	return "", false
}

func (te *TokenEncoder) resolveTokenModifiers(tokModifiers []lang.SemanticTokenModifier) semtok.TokenModifiers {
	modifiers := make(semtok.TokenModifiers, 0)

	for _, modifier := range tokModifiers {
		if modifier == lang.TokenModifierDependent {
			if te.tokenModifierSupported(string(lang.TokenModifierDependent)) {
				modifiers = append(modifiers, semtok.TokenModifier(lang.TokenModifierDependent))
				continue
			}
			if te.tokenModifierSupported(string(semtok.TokenModifierDefaultLibrary)) {
				modifiers = append(modifiers, semtok.TokenModifierDefaultLibrary)
				continue
			}
			continue
		}

		if te.tokenModifierSupported(string(modifier)) {
			modifiers = append(modifiers, semtok.TokenModifier(modifier))
		}
	}

	return modifiers
}

func (te *TokenEncoder) firstSupportedTokenType(tokenTypes ...semtok.TokenType) (semtok.TokenType, bool) {
	for _, tokenType := range tokenTypes {
		if te.tokenTypeSupported(string(tokenType)) {
			return tokenType, true
		}
	}
	return "", false
}

func (te *TokenEncoder) tokenTypeSupported(tokenType string) bool {
	return sliceContains(te.ClientCaps.TokenTypes, tokenType)
}

func (te *TokenEncoder) tokenModifierSupported(tokenModifier string) bool {
	return sliceContains(te.ClientCaps.TokenModifiers, tokenModifier)
}

func sliceContains(slice []string, value string) bool {
	for _, val := range slice {
		if val == value {
			return true
		}
	}
	return false
}
