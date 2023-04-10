// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package semtok

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
