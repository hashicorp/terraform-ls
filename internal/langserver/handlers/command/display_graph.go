// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

const displayGraphVersion = 0

type displayGraphResponse struct {
	FormatVersion int    `json:"v"`
	Nodes         []node `json:"nodes"`
	Edges         []edge `json:"edges"`
}

type node struct {
	lsp.Location
	Type   string   `json:"type"`
	Labels []string `json:"labels"`
}

type edge struct {
	From edgeNode `json:"from"`
	To   edgeNode `json:"to"`
}

type edgeNode = lsp.Location

func (h *CmdHandler) DisplayGraphHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	response := newDisplayGraphResponse()

	return response, nil
}

func newDisplayGraphResponse() displayGraphResponse {
	return displayGraphResponse{
		FormatVersion: displayGraphVersion,
		Nodes:         make([]node, 0),
		Edges:         make([]edge, 0),
	}
}
