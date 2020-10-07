package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/stretchr/testify/mock"
)

const intializeResponse = `{
	"jsonrpc": "2.0",
	"id": 1,
	"result": {
		"capabilities": {
			"textDocumentSync": {
				"openClose": true,
				"change": 2
			},
			"completionProvider": {},
			"documentSymbolProvider":true,
			"documentFormattingProvider":true,
			"executeCommandProvider": {
				"commands": ["rootmodule"]
			}
		}
	}
}`

func TestInitalizeAndShutdown(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			TempDir(t).Dir(): {TfExecFactory: validTfMockCalls()},
		}}))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())}, intializeResponse)
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
			TempDir(t).Dir(): {TfExecFactory: validTfMockCalls()},
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
	}`, TempDir(t).URI())}, intializeResponse)

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

func validTfMockCalls() exec.ExecutorFactory {
	return exec.NewMockExecutor([]*mock.Call{
		{
			Method:        "Version",
			Repeatability: 1,
			Arguments: []interface{}{
				mock.AnythingOfType(""),
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
				mock.AnythingOfType(""),
			},
			ReturnArguments: []interface{}{
				&tfjson.ProviderSchemas{FormatVersion: "0.1"},
				nil,
			},
		},
	})
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
