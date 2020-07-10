package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/langserver"
)

func TestInitalizeAndShutdown(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			TempDir(t).Dir(): {TerraformExecQueue: validTfMockCalls()},
		}}))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())}, `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"capabilities": {
				"textDocumentSync": {
					"openClose": true,
					"change": 2
				},
				"completionProvider": {},
				"documentFormattingProvider":true
			}
		}
	}`)
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`},
		`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": null
	}`)
}

func TestEOF(t *testing.T) {
	ms := newMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			TempDir(t).Dir(): {TerraformExecQueue: validTfMockCalls()},
		}})
	ls := langserver.NewLangServerMock(t, ms.new)
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())}, `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"capabilities": {
				"textDocumentSync": {
					"openClose": true,
					"change": 2
				},
				"completionProvider": {},
				"documentFormattingProvider":true
			}
		}
	}`)

	ls.CloseClientStdout(t)

	// Session is stopped after all other operations stop
	// which may take some time
	time.Sleep(1 * time.Millisecond)

	if !ms.StopFuncCalled() {
		t.Fatal("Expected session to stop on EOF")
	}
	if ls.StopFuncCalled() {
		t.Fatal("Expected server not to stop on EOF")
	}
}

func validTfMockCalls() *exec.MockQueue {
	return &exec.MockQueue{
		Q: []*exec.MockItem{
			{
				Args:   []string{"version", "-no-color"},
				Stdout: "Terraform v0.12.0\n",
			},
			{
				Args:   []string{"providers", "-no-color", "schema", "-json"},
				Stdout: "{\"format_version\":\"0.1\"}\n",
			},
		},
	}
}

func TestMain(m *testing.M) {
	if v := os.Getenv("TF_LS_MOCK"); v != "" {
		os.Exit(exec.ExecuteMockData(v))
		return
	}

	os.Exit(m.Run())
}

func TempDir(t *testing.T) lsp.FileHandler {
	tmpDir := filepath.Join(os.TempDir(), "terraform-ls", t.Name())

	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return lsp.FileHandlerFromDirPath(tmpDir)
		}
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
	})

	return lsp.FileHandlerFromDirPath(tmpDir)
}

func InitPluginCache(t *testing.T, dir string) {
	pluginCacheDir := filepath.Join(dir, ".terraform", "plugins")
	err := os.MkdirAll(pluginCacheDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(pluginCacheDir, "selections.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}
