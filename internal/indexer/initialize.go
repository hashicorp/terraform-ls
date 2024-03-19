package indexer

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) Initialize(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	err := idx.recordStores.TerraformVersions.AddIfNotExists()
	if err != nil {
		return nil, err
	}

	id, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			return module.GetInstalledTerraformVersion(ctx, idx.recordStores.TerraformVersions, modHandle.Path())
		},
		Type: op.OpTypeGetInstalledTerraformVersion.String(),
	})

	return job.IDs{id}, err
}
