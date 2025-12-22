// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_workspaceExecuteCommand_displayGraph_basic(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	InitPluginCache(t, tmpDir.Path())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
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
			"text": "variable \"region\" {\n  type = string\n}\n\nresource \"aws_instance\" \"example\" {\n  ami           = \"ami-0c55b159cbfafe1d0\"\n  instance_type = var.instance_type\n}\n\nvariable \"instance_type\" {\n  type = string\n}",
			"uri": %q
		}
	}`, testFileURI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q,
		"arguments": ["uri=%s"]
	}`, cmd.Name("terraform.display-graph"), tmpDir.Path()+"/main.tf")}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 3,
		"result": {
			"v": 0,
			"nodes": [
				{
					"id": 0,
					"uri": "file://%s",
					"range": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 17}
					},
					"type": "variable",
					"labels": ["region"]
				},
				{
					"id": 1,
					"uri": "file://%s",
					"range": {
						"start": {"line": 4, "character": 0},
						"end": {"line": 4, "character": 33}
					},
					"type": "resource",
					"labels": ["aws_instance", "example"]
				},
				{
					"id": 2,
					"uri": "file://%s",
					"range": {
						"start": {"line": 9, "character": 0},
						"end": {"line": 9, "character": 24}
					},
					"type": "variable",
					"labels": ["instance_type"]
				}
			],
			"edges": [
				{
					"from": 2,
					"to": 1
				}
			]
		}
	}`, tmpDir.Path()+"/main.tf", tmpDir.Path()+"/main.tf", tmpDir.Path()+"/main.tf"))
}

func TestLangServer_workspaceExecuteCommand_displayGraph_missingUri(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	InitPluginCache(t, tmpDir.Path())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
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
			"text": "provider \"github\" {}",
			"uri": %q
		}
	}`, testFileURI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q
	}`, cmd.Name("terraform.display-graph"))}, jrpc2.InvalidParams.Err())
}