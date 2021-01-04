package lsp

import (
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

type SemanticTokensClientCapabilities struct {
	lsp.SemanticTokensClientCapabilities
}

func (c SemanticTokensClientCapabilities) FullRequest() bool {
	switch full := c.Requests.Full.(type) {
	case bool:
		return full
	case map[string]interface{}:
		return true
	}
	return false
}
