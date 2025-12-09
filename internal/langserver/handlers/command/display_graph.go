// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
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

	docUri, ok := args.GetString("uri")
	if !ok || docUri == "" {
		return response, fmt.Errorf("%w: expected uri argument to be set", jrpc2.InvalidParams.Err())
	}

	dh := ilsp.HandleFromDocumentURI(lsp.DocumentURI(uri.FromPath(docUri)))
	doc, err := h.StateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return response, err
	}

	path := lang.Path{
		Path:       dh.Dir.Path(),
		LanguageID: doc.LanguageID,
	}

	pathDecoder, err := h.Decoder.Path(path)
	if err != nil {
		return response, err
	}

	nodes, err := getNodes(pathDecoder, path)
	if err != nil {
		return response, err
	}

	edges, err := getEdges(pathDecoder, path, h.Decoder)
	if err != nil {
		return response, err
	}

	response.Nodes = nodes
	response.Edges = edges
	return response, nil
}

func newDisplayGraphResponse() displayGraphResponse {
	return displayGraphResponse{
		FormatVersion: displayGraphVersion,
		Nodes:         make([]node, 0),
		Edges:         make([]edge, 0),
	}
}

func getNodes(pathDecoder *decoder.PathDecoder, path lang.Path) ([]node, error) {
	nodes := make([]node, 0)
	for _, file := range pathDecoder.Files() {
		body := file.Body.(*hclsyntax.Body)
		for _, block := range body.Blocks {
			nodes = append(nodes,
				node{
					Type:     block.Type,
					Labels:   block.Labels,
					Location: pathRangetoLocation(path, block.DefRange())})
		}

	}
	return nodes, nil
}

func getEdges(pathDecoder *decoder.PathDecoder, path lang.Path, decoder *decoder.Decoder) ([]edge, error) {
	edges := make([]edge, 0)
	refTargets := pathDecoder.RefTargets()
	seen := make(map[string]bool)

	for _, refTarget := range refTargets {
		if refTarget.DefRangePtr != nil {
			fromEdge := pathRangetoLocation(path, *refTarget.DefRangePtr)
			origins := decoder.ReferenceOriginsByTarget(context.Background(), refTarget, path)
			for _, refOrigin := range origins {
				edge := edge{
					From: fromEdge,
					To:   pathRangetoLocation(path, refOrigin.RootBlockRange),
				}
				edgeKey := edgeKey(edge)
				if isASelfEdge(edge) || isSeenEdge(&seen, edgeKey) {
					continue
				}

				edges = append(edges, edge)
				seen[edgeKey] = true
			}

		}
	}
	return edges, nil
}

func edgeKey(e edge) string {
	return edgeNodeKey(e.From) + "->" + edgeNodeKey(e.To)
}

func edgeNodeKey(e edgeNode) string {
	return fmt.Sprintf("%s#%d:%d#%d:%d", e.URI, e.Range.Start.Line, e.Range.Start.Character, e.Range.End.Line, e.Range.End.Character)
}

func isSeenEdge(seen *map[string]bool, edgeKey string) bool {
	_, ok := (*seen)[edgeKey]
	return ok
}

func isASelfEdge(edge edge) bool {
	return edge.From == edge.To
}

func pathRangetoLocation(path lang.Path, rng hcl.Range) lsp.Location {
	return lsp.Location{
		URI:   lsp.DocumentURI(uri.FromPath(filepath.Join(path.Path, rng.Filename))),
		Range: ilsp.HCLRangeToLSP(rng),
	}
}
