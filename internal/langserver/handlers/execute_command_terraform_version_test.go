// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/uri"
	"github.com/hashicorp/terraform-ls/internal/walker"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_workspaceExecuteCommand_terraformVersion_basic(t *testing.T) {
	modDir := t.TempDir()
	modUri := uri.FromPath(modDir)

	s, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	err = s.Modules.Add(modDir)
	if err != nil {
		t.Fatal(err)
	}

	metadata := &tfmod.Meta{
		Path:             modDir,
		CoreRequirements: testConstraint(t, "~> 0.15"),
	}

	err = s.Modules.UpdateMetadata(modDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	ver, err := version.NewVersion("1.1.0")
	if err != nil {
		t.Fatal(err)
	}

	err = s.Modules.UpdateTerraformAndProviderVersions(modDir, ver, map[tfaddr.Provider]*version.Version{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				modDir: validTfMockCalls(),
			},
		},
		StateStore:      s,
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
	}`, modUri)})
	waitForWalkerPath(t, s, wc, document.DirHandleFromURI(modUri))
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q,
		"arguments": ["uri=%s"]
	}`, cmd.Name("module.terraform"), modUri)}, `{
		"jsonrpc": "2.0",
		"id": 2,
		"result": {
			"v": 0,
			"required_version": "~\u003e 0.15",
			"discovered_version": "1.1.0"
		}
	}`)
}
