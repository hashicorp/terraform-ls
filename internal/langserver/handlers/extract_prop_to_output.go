// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) ExtractPropToOutput(ctx context.Context, params lsp.CodeActionParams) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)

	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return edits, err
	}

	mod, err := svc.stateStore.Modules.ModuleByPath(dh.Dir.Path())
	if err != nil {
		return edits, err
	}

	file, ok := mod.ParsedModuleFiles.AsMap()[dh.Filename]
	if !ok {
		return edits, err
	}

	pos, err := ilsp.HCLPositionFromLspPosition(params.Range.Start, doc)
	if err != nil {
		return edits, err
	}

	blocks := file.BlocksAtPos(pos)
	if len(blocks) > 1 {
		return edits, fmt.Errorf("found more than one block at pos: %v", pos)
	}
	if len(blocks) == 0 {
		return edits, fmt.Errorf("can not find block at position %v", pos)
	}

	attr := file.AttributeAtPos(pos)
	if attr == nil {
		return edits, fmt.Errorf("can not find attribute at position %v", pos)
	}

	tfAddr := append(blocks[0].Labels, attr.Name)

	insertPos := lsp.Position{
		Line:      uint32(len(doc.Lines)),
		Character: uint32(len(doc.Lines)),
	}

	edits = append(edits, lsp.TextEdit{
		Range: lsp.Range{
			Start: insertPos,
			End:   insertPos,
		},
		NewText: outputBlock(strings.Join(tfAddr, "_"), strings.Join(tfAddr, ".")),
	})
	return edits, nil
}

func outputBlock(name, tfAddr string) string {
	f := hclwrite.NewFile()
	b := hclwrite.NewBlock("output", []string{name})
	b.Body().SetAttributeRaw("value", hclwrite.TokensForIdentifier(tfAddr))
	f.Body().AppendNewline()
	f.Body().AppendBlock(b)
	return string(f.Bytes())
}
