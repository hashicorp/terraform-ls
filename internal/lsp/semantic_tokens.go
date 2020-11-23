package lsp

import (
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

type semanticTokensFull struct {
	Delta bool `json:"delta,omitempty"`
}

type SemanticTokensClientCapabilities struct {
	lsp.SemanticTokensClientCapabilities
}

func (c SemanticTokensClientCapabilities) FullRequest() bool {
	switch full := c.Requests.Full.(type) {
	case bool:
		return full
	case semanticTokensFull:
		return true
	}
	return false
}
