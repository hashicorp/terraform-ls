package rootmodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TestNewRootModuleManagerMock_noMocks(t *testing.T) {
	f := NewRootModuleManagerMock(nil)
	rmm := f()
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
				TerraformExecQueue: &exec.MockQueue{
					Q: []*exec.MockItem{
						{
							Args:   []string{"version"},
							Stdout: "Terraform v0.12.0\n",
						},
						{
							Args:   []string{"providers", "schema", "-json"},
							Stdout: "{\"format_version\":\"0.1\"}\n",
						},
					},
				},
			},
		}})
	rmm := f()
	_, err := rmm.AddAndStartLoadingRootModule(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	if v := os.Getenv("TF_LS_MOCK"); v != "" {
		os.Exit(exec.ExecuteMockData(v))
		return
	}

	os.Exit(m.Run())
}
