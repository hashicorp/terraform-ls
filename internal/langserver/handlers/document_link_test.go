// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func TestDocumentLink_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())
	err := ioutil.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("provider \"test\" {\n\n}\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.1")),
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
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/documentLink",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": [
				{
					"range": {
						"start": {
							"line": 0,
							"character": 9
						},
						"end": {
							"line": 0,
							"character": 15
						}
					},
					"target": "https://registry.terraform.io/providers/test/test/latest/docs?utm_content=documentLink\u0026utm_source=terraform-ls"
				}
			]
		}`)
}
