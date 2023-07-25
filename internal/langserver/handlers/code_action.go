// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (svc *service) TextDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) []lsp.CodeAction {
	ca, err := svc.textDocumentCodeAction(ctx, params)
	if err != nil {
		svc.logger.Printf("code action failed: %s", err)
	}

	return ca
}

func (svc *service) textDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	var ca []lsp.CodeAction

	// For action definitions, refer to https://code.visualstudio.com/api/references/vscode-api#CodeActionKind
	// We only support format type code actions at the moment, and do not want to format without the client asking for
	// them, so exit early here if nothing is requested.
	// if len(params.Context.Only) == 0 {
	// 	svc.logger.Printf("No code action requested, exiting")
	// 	return ca, nil
	// }

	for _, o := range params.Context.Only {
		svc.logger.Printf("Code actions requested: %q", o)
	}

	// wantedCodeActions := ilsp.SupportedCodeActions.Only(params.Context.Only)
	// if len(wantedCodeActions) == 0 {
	// 	return nil, fmt.Errorf("could not find a supported code action to execute for %s, wanted %v",
	// 		params.TextDocument.URI, params.Context.Only)
	// }

	// The Only field of the context specifies which code actions the client wants.
	// If Only is empty, assume that the client wants all of the non-explicit code actions.
	var wantedCodeActions map[lsp.CodeActionKind]bool

	if len(params.Context.Only) == 0 {
		wantedCodeActions = ilsp.SupportedCodeActions // TODO! filter by type
	} else {
		wantedCodeActions = ilsp.SupportedCodeActions.Only(params.Context.Only)
	}

	svc.logger.Printf("Code actions supported: %v", wantedCodeActions)

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)

	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return ca, err
	}

	// mod, err := svc.modStore.ModuleByPath(dh.Dir.Path())
	// if err != nil {
	// 	return ca, err
	// }

	codeActionRange, err := ilsp.HCLRangeFromLspRange(params.Range, doc)
	if err != nil {
		return ca, err
	}
	svc.logger.Printf("CODE ACTION RANGE %#v", codeActionRange)

	for action := range wantedCodeActions {
		switch action {
		case lsp.QuickFix:
			// modDiags := mod.ModuleValidationDiagnostics.AutoloadedOnly().AsMap()
			contextDiags := params.Context.Diagnostics

			for _, diag := range contextDiags {

				// for _, d := range diags {
				kind := parseDiagnostic(diag)
				if kind == Unknown {
					continue
				}

				// if !d.Subject.ContainsPos(codeActionRange.Start) {
				// 	continue
				// }
				edit := buildEditForKind(kind, params.TextDocument.URI, diag.Range)
				if edit == nil {
					continue
				}
				ca = append(ca, lsp.CodeAction{
					Title:       "Fix issue",
					Kind:        action,
					IsPreferred: true,
					// Diagnostics: ilsp.HCLDiagsToLSP(hcl.Diagnostics{d}, "early validation"),
					Edit: *edit,
				})
				// }
			}

		case ilsp.SourceFormatAllTerraform:
			tfExec, err := module.TerraformExecutorForModule(ctx, dh.Dir.Path())
			if err != nil {
				return ca, errors.EnrichTfExecError(err)
			}

			edits, err := svc.formatDocument(ctx, tfExec, doc.Text, dh)
			if err != nil {
				return ca, err
			}

			ca = append(ca, lsp.CodeAction{
				Title: "Format Document",
				Kind:  action,
				Edit: lsp.WorkspaceEdit{
					Changes: map[lsp.DocumentURI][]lsp.TextEdit{
						lsp.DocumentURI(dh.FullURI()): edits,
					},
				},
			})
		}
	}

	return ca, nil
}

type DiagnosticKind int64

const (
	Unknown DiagnosticKind = iota
	ExtraneousLabel
	MissingLabel
)

func parseDiagnostic(diag lsp.Diagnostic) DiagnosticKind {
	// if diag == nil {
	// 	return Unknown
	// }

	msg := diag.Message
	if strings.HasPrefix(msg, "Missing name for") {
		return MissingLabel
	} else if strings.HasPrefix(msg, "Extraneous label for") {
		return ExtraneousLabel
	}

	return Unknown
}

func buildEditForKind(kind DiagnosticKind, uri lsp.DocumentURI, rng lsp.Range) *lsp.WorkspaceEdit {
	if kind == ExtraneousLabel {
		return &lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				uri: {
					{
						// Range:   ilsp.HCLRangeToLSP(*d.Subject),
						Range:   rng,
						NewText: "",
					},
				},
			},
		}
	}
	if kind == MissingLabel {
		return &lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				uri: {
					{
						// Range:   ilsp.HCLRangeToLSP(*d.Subject),
						Range:   rng,
						NewText: `"" {`,
					},
				},
			},
		}
	}

	return nil
}
