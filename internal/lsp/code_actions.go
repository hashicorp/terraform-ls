package lsp

import (
	"sort"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

const (
	SourceFormatAll            = "source.formatAll"
	SourceFormatAllTerraformLs = "source.formatAll.terraform-ls"
)

type CodeActions map[lsp.CodeActionKind]bool

var (
	SupportedCodeActions = CodeActions{
		lsp.Source:                 true,
		lsp.SourceFixAll:           true,
		SourceFormatAll:            true,
		SourceFormatAllTerraformLs: true,
	}
)

func (c CodeActions) AsSlice() []lsp.CodeActionKind {
	s := make([]lsp.CodeActionKind, 0)
	for v := range c {
		s = append(s, v)
	}

	sort.SliceStable(s, func(i, j int) bool {
		return string(s[i]) < string(s[j])
	})
	return s
}

func (ca CodeActions) Only(only []lsp.CodeActionKind) CodeActions {
	// if only is empty, assume that the client wants all code actions
	// else build mapping of requested and determine if supported
	if len(only) == 0 {
		return ca
	}

	wanted := make(CodeActions, 0)
	for _, kind := range only {
		if v, ok := ca[kind]; ok {
			wanted[kind] = v
		}
	}

	return wanted
}
