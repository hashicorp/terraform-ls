// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func Links(links []lang.Link, caps lsp.DocumentLinkClientCapabilities) []lsp.DocumentLink {
	docLinks := make([]lsp.DocumentLink, len(links))

	for i, link := range links {
		tooltip := ""
		if caps.TooltipSupport {
			tooltip = link.Tooltip
		}
		docLinks[i] = lsp.DocumentLink{
			Range:   HCLRangeToLSP(link.Range),
			Target:  link.URI,
			Tooltip: tooltip,
		}
	}

	return docLinks
}
