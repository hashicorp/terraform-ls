package rootmodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TestNewRootModuleManagerMock_noMocks(t *testing.T) {
	f := NewRootModuleManagerMock(map[string]*RootModuleMock{})
	rmm := f(context.Background())
	err := rmm.AddRootModule("any-path")
	if err == nil {
		t.Fatal("expected unmocked path addition to fail")
	}
}

func TestNewRootModuleManagerMock_mocks(t *testing.T) {
	tmpDir := filepath.Clean(os.TempDir())

	f := NewRootModuleManagerMock(map[string]*RootModuleMock{
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
	})
	rmm := f(context.Background())
	err := rmm.AddRootModule(tmpDir)
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
