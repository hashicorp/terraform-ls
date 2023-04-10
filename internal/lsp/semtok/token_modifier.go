// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package semtok

import "math"

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
