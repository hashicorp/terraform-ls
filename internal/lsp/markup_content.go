// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func markupContent(content lang.MarkupContent, mdSupported bool) lsp.MarkupContent {
	value := content.Value

	kind := lsp.PlainText
	if content.Kind == lang.MarkdownKind {
		if mdSupported {
			kind = lsp.Markdown
		} else {
			value = mdplain.Clean(value)
		}
	}

	return lsp.MarkupContent{
		Kind:  kind,
		Value: value,
	}
}
