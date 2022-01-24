package module

import (
	"context"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

type ModuleManagerMockInput struct {
	Logger         *log.Logger
	TerraformCalls *exec.TerraformMockCalls
}

func NewModuleManagerMock(input *ModuleManagerMockInput) ModuleManagerFactory {
	var logger *log.Logger
	var tfCalls *exec.TerraformMockCalls

	if input != nil {
		logger = input.Logger
		tfCalls = input.TerraformCalls
	}

	return func(ctx context.Context, fs ReadOnlyFS, ds DocumentStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore) ModuleManager {
		if tfCalls != nil {
			ctx = exec.WithExecutorFactory(ctx, exec.NewMockExecutor(tfCalls))
			ctx = exec.WithExecutorOpts(ctx, &exec.ExecutorOpts{
				ExecPath: "tf-mock",
			})
		}

		mm := NewSyncModuleManager(ctx, fs, ds, ms, pss)

		if logger != nil {
			mm.SetLogger(logger)
		}

		return mm
	}
}

func validTfMockCalls(repeatability int) []*mock.Call {
	return []*mock.Call{
		{
			Method:        "Version",
			Repeatability: repeatability,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.cancelCtx"),
			},
			ReturnArguments: []interface{}{
				version.Must(version.NewVersion("0.12.0")),
				nil,
				nil,
			},
		},
		{
			Method:        "GetExecPath",
			Repeatability: repeatability,
			ReturnArguments: []interface{}{
				"",
			},
		},
		{
			Method:        "ProviderSchemas",
			Repeatability: repeatability,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.cancelCtx"),
			},
			ReturnArguments: []interface{}{
				&tfjson.ProviderSchemas{
					FormatVersion: "0.1",
					Schemas: map[string]*tfjson.ProviderSchema{
						"test": {
							ConfigSchema: &tfjson.Schema{},
						},
					},
				},
				nil,
			},
		},
	}
}
