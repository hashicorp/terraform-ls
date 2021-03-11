package lsp

import (
	"math"
)

type TokenType string
type TokenTypes []TokenType

func (tt TokenTypes) AsStrings() []string {
	types := make([]string, len(tt))

	for i, tokenType := range tt {
		types[i] = string(tokenType)
	}

	return types
}

func (tt TokenTypes) Index(tokenType TokenType) int {
	for i, t := range tt {
		if t == tokenType {
			return i
		}
	}
	return -1
}

type TokenModifier string
type TokenModifiers []TokenModifier

func (tm TokenModifiers) AsStrings() []string {
	modifiers := make([]string, len(tm))

	for i, tokenModifier := range tm {
		modifiers[i] = string(tokenModifier)
	}

	return modifiers
}

func (tm TokenModifiers) BitMask(declaredModifiers TokenModifiers) int {
	bitMask := 0b0

	for i, modifier := range tm {
		if isDeclared(modifier, declaredModifiers) {
			bitMask |= int(math.Pow(2, float64(i)))
		}
	}

	return bitMask
}

func isDeclared(mod TokenModifier, declaredModifiers TokenModifiers) bool {
	for _, dm := range declaredModifiers {
		if mod == dm {
			return true
		}
	}
	return false
}

const (
	// Types predefined in LSP spec
	TokenTypeClass         TokenType = "class"
	TokenTypeComment       TokenType = "comment"
	TokenTypeEnum          TokenType = "enum"
	TokenTypeEnumMember    TokenType = "enumMember"
	TokenTypeEvent         TokenType = "event"
	TokenTypeFunction      TokenType = "function"
	TokenTypeInterface     TokenType = "interface"
	TokenTypeKeyword       TokenType = "keyword"
	TokenTypeMacro         TokenType = "macro"
	TokenTypeMethod        TokenType = "method"
	TokenTypeModifier      TokenType = "modifier"
	TokenTypeNamespace     TokenType = "namespace"
	TokenTypeNumber        TokenType = "number"
	TokenTypeOperator      TokenType = "operator"
	TokenTypeParameter     TokenType = "parameter"
	TokenTypeProperty      TokenType = "property"
	TokenTypeRegexp        TokenType = "regexp"
	TokenTypeString        TokenType = "string"
	TokenTypeStruct        TokenType = "struct"
	TokenTypeType          TokenType = "type"
	TokenTypeTypeParameter TokenType = "typeParameter"
	TokenTypeVariable      TokenType = "variable"

	// Modifiers predefined in LSP spec
	TokenModifierDeclaration    TokenModifier = "declaration"
	TokenModifierDefinition     TokenModifier = "definition"
	TokenModifierReadonly       TokenModifier = "readonly"
	TokenModifierStatic         TokenModifier = "static"
	TokenModifierDeprecated     TokenModifier = "deprecated"
	TokenModifierAbstract       TokenModifier = "abstract"
	TokenModifierAsync          TokenModifier = "async"
	TokenModifierModification   TokenModifier = "modification"
	TokenModifierDocumentation  TokenModifier = "documentation"
	TokenModifierDefaultLibrary TokenModifier = "defaultLibrary"
)

// Registering types which are actually in use and known
// to be registered by VS Code by default, see https://git.io/JIeuV
var (
	serverTokenTypes = TokenTypes{
		TokenTypeType,
		TokenTypeString,
		TokenTypeProperty,
		TokenTypeKeyword,
		TokenTypeNumber,
		TokenTypeParameter,
	}
	serverTokenModifiers = TokenModifiers{
		TokenModifierDeprecated,
		TokenModifierModification,
	}
)

func TokenTypesLegend(clientSupported []string) TokenTypes {
	legend := make(TokenTypes, 0)

	// Filter only supported token types
	for _, tokenType := range serverTokenTypes {
		if sliceContains(clientSupported, string(tokenType)) {
			legend = append(legend, TokenType(tokenType))
		}
	}

	return legend
}

func TokenModifiersLegend(clientSupported []string) TokenModifiers {
	legend := make(TokenModifiers, 0)

	// Filter only supported token modifiers
	for _, modifier := range serverTokenModifiers {
		if sliceContains(clientSupported, string(modifier)) {
			legend = append(legend, TokenModifier(modifier))
		}
	}

	return legend
}

func sliceContains(slice []string, value string) bool {
	for _, val := range slice {
		if val == value {
			return true
		}
	}
	return false
}
