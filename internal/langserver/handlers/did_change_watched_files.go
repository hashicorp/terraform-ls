package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

func (svc *service) DidChangeWatchedFiles(ctx context.Context, params lsp.DidChangeWatchedFilesParams) error {
	var ids job.IDs

	for _, change := range params.Changes {
		uri := string(change.URI)
		_, err := os.Stat(uri)
		if err != nil {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type:    lsp.Warning,
				Message: fmt.Sprintf("Unable to update: %s, Failed to open directory", uri),
			})
			continue
		}

		// `uri` can either be a file or a director baed on the spec.
		// We're not making any assumptions on the above and passing
		// the uri as the module path itself for validation.
		//
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#workspace_didChangeWatchedFiles
		if !ast.IsModuleFilename(uri) && !ast.IsVarsFilename(uri) {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type:    lsp.Warning,
				Message: fmt.Sprintf("Unable to update file: %s, filetype not supported", uri),
			})
			continue
		}

		// only handle `Changed` event type
		if change.Type == protocol.Changed {
			dh := ilsp.HandleFromDocumentURI(change.URI)

			// check existence
			_, err = svc.modStore.ModuleByPath(dh.Dir.Path())
			if err != nil {
				continue
			}

			jobIds, err := svc.parseAndDecodeModule(dh.Dir)
			if err != nil {
				continue
			}

			ids = append(ids, jobIds...)

		}

	}

	// wait for all jobs (slowest usually) to complete
	err := svc.stateStore.JobStore.WaitForJobs(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}
