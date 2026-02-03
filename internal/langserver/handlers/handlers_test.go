// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func initializeResponse(t *testing.T, commandPrefix string) string {
	jsonArray, err := json.Marshal(cmdHandlers(&service{}).Names(commandPrefix))
	if err != nil {
		t.Fatal(err)
	}

	return fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"capabilities": {
				"textDocumentSync": {
					"openClose": true,
					"change": 2,
					"save": {}
				},
				"completionProvider": {
					"triggerCharacters": [".", "["],
					"resolveProvider": true
				},
				"hoverProvider": true,
				"signatureHelpProvider": {
					"triggerCharacters":["(",","]
				},
				"declarationProvider": true,
				"definitionProvider": true,
				"referencesProvider": true,
				"documentSymbolProvider": true,
				"codeActionProvider": {
					"codeActionKinds": ["source.formatAll.terraform"]
				},
				"codeLensProvider": {},
				"documentLinkProvider": {},
				"workspaceSymbolProvider": true,
				"documentFormattingProvider": true,
				"executeCommandProvider": {
					"commands": %s,
					"workDoneProgress":true
				},
				"semanticTokensProvider": {
					"legend": {
						"tokenTypes": [],
						"tokenModifiers": []
					}
				},
				"workspace": {
					"workspaceFolders": {
						"supported": true,
						"changeNotifications": "workspace/didChangeWorkspaceFolders"
					},
					"fileOperations": {}
				},
				"experimental": {
					"referenceCountCodeLens": false,
					"refreshModuleProviders": false,
					"refreshModuleCalls": false,
					"refreshTerraformVersion": false
				}
			},
			"serverInfo": {
				"name": "terraform-ls",
				"version": ""
			}
		}
	}`, string(jsonArray))
}

func waitForWalkerPath(t testOrBench, ss *state.StateStore, wc *walker.WalkerCollector, dir document.DirHandle) {
	ctx := context.Background()
	err := ss.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{dir})
	if err != nil {
		t.Fatal(err)
	}
	err = ss.JobStore.WaitForJobs(ctx, wc.JobIds()...)
	if err != nil {
		t.Fatal(err)
	}
	err = wc.ErrorOrNil()
	if err != nil {
		t.Fatal(err)
	}
}

func waitForAllJobs(t testOrBench, ss *state.StateStore) {
	ctx := context.Background()
	ids, err := ss.JobStore.ListAllJobs()
	if err != nil {
		t.Fatal(err)
	}
	err = ss.JobStore.WaitForJobs(ctx, ids...)
	if err != nil {
		t.Fatal(err)
	}
}

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func TestInitializeAndShutdown(t *testing.T) {
	tmpDir := TempDir(t)

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)}, initializeResponse(t, ""))
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`},
		`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": null
	}`)
}

func TestInitializeWithCommandPrefix(t *testing.T) {
	tmpDir := TempDir(t)

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345,
		"initializationOptions": {
			"commandPrefix": "1"
		}
	}`, tmpDir.URI)}, initializeResponse(t, "1"))
}

func TestEOF(t *testing.T) {
	tmpDir := TempDir(t)

	ms := newMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
			},
		},
	})
	ls := langserver.NewLangServerMock(t, ms.new)
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)}, initializeResponse(t, ""))

	ls.CloseClientStdout(t)

	// Session is stopped after all other operations stop
	// which may take some time
	time.Sleep(250 * time.Millisecond)

	if !ms.StopFuncCalled() {
		t.Fatal("Expected session to stop on EOF")
	}
	if ls.StopFuncCalled() {
		t.Fatal("Expected server not to stop on EOF")
	}
}

func validTfMockCalls() []*mock.Call {
	return []*mock.Call{
		{
			Method:        "Version",
			Repeatability: 1,
			Arguments: []interface{}{
				mock.AnythingOfType(""),
			},
			ReturnArguments: []interface{}{
				version.Must(version.NewVersion("0.12.0")),
				nil,
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

// TempDir creates a temporary directory containing the test name, as well any
// additional nested dir specified, use slash "/" to nest for more complex
// setups
//
//	ex: TempDir(t, "a/b", "c")
//	├── a
//	│   └── b
//	└── c
//
// The returned filehandler is the parent tmp dir
func TempDir(t *testing.T, nested ...string) document.DirHandle {
	tmpDir := filepath.Join(os.TempDir(), "terraform-ls", t.Name())
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, dir := range nested {
		err := os.MkdirAll(filepath.Join(tmpDir, filepath.FromSlash(dir)), 0755)
		if err != nil && !os.IsExist(err) {
			t.Fatal(err)
		}
	}

	return document.DirHandleFromPath(tmpDir)
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

	// create an empty file such that it's recognized as an indexable workspace
	f, err = os.Create(filepath.Join(dir, "empty.tf"))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}
