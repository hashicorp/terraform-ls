package rootmodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestNewRootModuleManagerMock_noMocks(t *testing.T) {
	f := NewRootModuleManagerMock(nil)
	rmm := f(filesystem.NewFilesystem())
	_, err := rmm.AddAndStartLoadingRootModule(context.Background(), "any-path")
	if err == nil {
		t.Fatal("expected unmocked path addition to fail")
	}
}

func TestNewRootModuleManagerMock_mocks(t *testing.T) {
	tmpDir := filepath.Clean(os.TempDir())

	f := NewRootModuleManagerMock(&RootModuleManagerMockInput{
		RootModules: map[string]*RootModuleMock{
			tmpDir: {
				TfExecFactory: validTfMockCalls(t, tmpDir),
			},
		}})
	rmm := f(filesystem.NewFilesystem())
	_, err := rmm.AddAndStartLoadingRootModule(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}
}

func validTfMockCalls(t *testing.T, workDir string) exec.ExecutorFactory {
	return exec.NewMockExecutor([]*mock.Call{
		{
			Method:        "Version",
			Repeatability: 1,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.emptyCtx"),
			},
			ReturnArguments: []interface{}{
				version.Must(version.NewVersion("0.12.0")),
				nil,
			},
		},
		{
			Method:        "GetExecPath",
			Repeatability: 1,
			ReturnArguments: []interface{}{
				"",
			},
		},
		{
			Method:        "ProviderSchemas",
			Repeatability: 1,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.emptyCtx"),
			},
			ReturnArguments: []interface{}{
				&tfjson.ProviderSchemas{FormatVersion: "0.1"},
				nil,
			},
		},
	})
}
