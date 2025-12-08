// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package utm

import (
	"context"

	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
)

const UtmSource = "terraform-ls"

func UtmMedium(ctx context.Context) string {
	clientName, ok := ilsp.ClientName(ctx)
	if ok {
		return clientName
	}

	return ""
}
