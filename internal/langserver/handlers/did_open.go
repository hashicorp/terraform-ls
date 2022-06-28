package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (svc *service) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	docURI := string(params.TextDocument.URI)

	// URIs are always checked during initialize request, but
	// we still allow single-file mode, therefore invalid URIs
	// can still land here, so we check for those.
	if !uri.IsURIValid(docURI) {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("Ignoring workspace folder (unsupport or invalid URI) %s."+
				" This is most likely bug, please report it.", docURI),
		})
		return fmt.Errorf("invalid URI: %s", docURI)
	}

	dh := document.HandleFromURI(docURI)

	err := svc.stateStore.DocumentStore.OpenDocument(dh, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), []byte(params.TextDocument.Text))
	if err != nil {
		return err
	}

	mod, err := svc.modStore.ModuleByPath(dh.Dir.Path())
	if err != nil {
		if state.IsModuleNotFound(err) {
			err = svc.modStore.Add(dh.Dir.Path())
			if err != nil {
				return err
			}
			mod, err = svc.modStore.ModuleByPath(dh.Dir.Path())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	svc.logger.Printf("opened module: %s", mod.Path)

	// We reparse because the file being opened may not match
	// (originally parsed) content on the disk
	// TODO: Do this only if we can verify the file differs?
	modHandle := document.DirHandleFromPath(mod.Path)
	jobIds, err := svc.parseAndDecodeModule(modHandle)
	if err != nil {
		return err
	}

	if mod.TerraformVersionState == op.OpStateUnknown {
		jobId, err := svc.stateStore.JobStore.EnqueueJob(job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)
				return module.GetTerraformVersion(ctx, svc.modStore, mod.Path)
			},
			Type: op.OpTypeGetTerraformVersion.String(),
		})
		if err != nil {
			return err
		}
		jobIds = append(jobIds, jobId)
	}

	if svc.singleFileMode {
		err = svc.stateStore.WalkerPaths.EnqueueDir(modHandle)
		if err != nil {
			return err
		}
	}

	return svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)
}

func (svc *service) parseAndDecodeModule(modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := svc.stateStore.JobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(svc.fs, svc.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
		Defer: func(ctx context.Context, jobErr error) job.IDs {
			ids, err := svc.decodeModule(ctx, modHandle)
			if err != nil {
				svc.logger.Printf("error: %s", err)
			}
			return ids
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	id, err = svc.stateStore.JobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(svc.fs, svc.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseVariables.String(),
		Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
			id, err := svc.stateStore.JobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeVarsReferences(ctx, svc.modStore, svc.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeVarsReferences.String(),
			})
			if err != nil {
				return
			}
			ids = append(ids, id)
			return
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}

func (svc *service) decodeModule(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := svc.stateStore.JobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.LoadModuleMetadata(svc.modStore, modHandle.Path())
		},
		Type: op.OpTypeLoadModuleMetadata.String(),
		Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
			id, err := svc.stateStore.JobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceTargets(ctx, svc.modStore, svc.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeReferenceTargets.String(),
			})
			if err != nil {
				return
			}
			ids = append(ids, id)

			id, err = svc.stateStore.JobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceOrigins(ctx, svc.modStore, svc.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeReferenceOrigins.String(),
			})
			if err != nil {
				return
			}
			ids = append(ids, id)

			_, err = svc.stateStore.JobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.GetModuleDataFromRegistry(svc.srvCtx, svc.registryClient,
						svc.modStore, svc.stateStore.RegistryModules, modHandle.Path())
				},
				Priority: job.LowPriority,
				Type:     op.OpTypeGetModuleDataFromRegistry.String(),
			})
			if err != nil {
				return
			}

			return
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}
