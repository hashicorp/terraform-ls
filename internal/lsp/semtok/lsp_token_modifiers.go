// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package semtok

var (
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
