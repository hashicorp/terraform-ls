package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (svc *service) DidChangeWatchedFiles(ctx context.Context, params lsp.DidChangeWatchedFilesParams) error {
	var ids job.IDs

	for _, change := range params.Changes {
		rawURI := string(change.URI)

		fullPath, err := uri.PathFromURI(rawURI)
		if err != nil {
			svc.logger.Printf("Unable to update %q: %s", rawURI, err)
			continue
		}

		fi, err := os.Stat(fullPath)
		if err != nil {
			svc.logger.Printf("Unable to update %q: %s ", fullPath, err)
			continue
		}

		// URI can either be a file or a directory based on the LSP spec.
		var dirHandle document.DirHandle
		if !fi.IsDir() {
			if !ast.IsModuleFilename(fi.Name()) && !ast.IsVarsFilename(fi.Name()) {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type: lsp.Warning,
					Message: fmt.Sprintf("Unable to update %q: filetype not supported. "+
						"This is likely a bug which should be reported.", fullPath),
				})
				continue
			}
			docHandle := document.HandleFromPath(fullPath)
			dirHandle = docHandle.Dir
		} else {
			dirHandle = document.DirHandleFromPath(fullPath)
		}

		if change.Type == protocol.Changed {
			_, err = svc.modStore.ModuleByPath(dirHandle.Path())
			if err != nil {
				continue
			}

			jobIds, err := svc.parseAndDecodeModule(dirHandle)
			if err != nil {
				continue
			}

			ids = append(ids, jobIds...)
		}
	}

	err := svc.stateStore.JobStore.WaitForJobs(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}
