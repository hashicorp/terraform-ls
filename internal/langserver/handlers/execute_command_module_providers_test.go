// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
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

func TestLangServer_workspaceExecuteCommand_moduleProviders_argumentError(t *testing.T) {
	rootDir := document.DirHandleFromPath(t.TempDir())

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				rootDir.Path(): validTfMockCalls(),
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
	}`, rootDir.URI)})
	waitForWalkerPath(t, ss, wc, rootDir)
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
	}`, fmt.Sprintf("%s/main.tf", rootDir.URI))})
	waitForAllJobs(t, ss)

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q
	}`, cmd.Name("module.providers"))}, jrpc2.InvalidParams.Err())
}

func TestLangServer_workspaceExecuteCommand_moduleProviders_basic(t *testing.T) {
	modDir := t.TempDir()
	modUri := uri.FromPath(modDir)

	s, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			modDir: validTfMockCalls(),
		},
	}
	fs := filesystem.NewFilesystem(s.DocumentStore)
	features, err := NewTestFeatures(eventBus, s, fs, mockCalls)
	if err != nil {
		t.Fatal(err)
	}

	err = features.Modules.Store.Add(modDir)
	if err != nil {
		t.Fatal(err)
	}
	err = features.RootModules.Store.Add(modDir)
	if err != nil {
		t.Fatal(err)
	}

	metadata := &tfmod.Meta{
		Path:             modDir,
		CoreRequirements: testConstraint(t, "~> 0.15"),
		ProviderRequirements: map[tfaddr.Provider]version.Constraints{
			newDefaultProvider("aws"):    testConstraint(t, "1.2.3"),
			newDefaultProvider("google"): testConstraint(t, ">= 2.0.0"),
		},
		ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
			{LocalName: "aws"}:    newDefaultProvider("aws"),
			{LocalName: "google"}: newDefaultProvider("google"),
		},
	}

	err = features.Modules.Store.UpdateMetadata(modDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	pVersions := map[tfaddr.Provider]*version.Version{
		newDefaultProvider("aws"):    version.Must(version.NewVersion("1.2.3")),
		newDefaultProvider("google"): version.Must(version.NewVersion("2.5.5")),
	}
	err = features.RootModules.Store.UpdateInstalledProviders(modDir, pVersions, nil)
	if err != nil {
		t.Fatal(err)
	}

	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls:  mockCalls,
		StateStore:      s,
		WalkerCollector: wc,
		Features:        features,
		EventBus:        eventBus,
		FileSystem:      fs,
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
	}`, cmd.Name("module.providers"), modUri)}, `{
		"jsonrpc": "2.0",
		"id": 2,
		"result": {
			"v": 0,
			"provider_requirements": {
				"registry.terraform.io/hashicorp/aws": {
					"display_name": "hashicorp/aws",
					"version_constraint":"1.2.3",
					"docs_link": "https://registry.terraform.io/providers/hashicorp/aws/latest?utm_content=workspace%2FexecuteCommand%2Fmodule.providers\u0026utm_source=terraform-ls"
				},
				"registry.terraform.io/hashicorp/google": {
					"display_name": "hashicorp/google",
					"version_constraint": "\u003e= 2.0.0",
					"docs_link": "https://registry.terraform.io/providers/hashicorp/google/latest?utm_content=workspace%2FexecuteCommand%2Fmodule.providers\u0026utm_source=terraform-ls"
				}
			},
			"installed_providers":{
				"registry.terraform.io/hashicorp/aws": "1.2.3",
				"registry.terraform.io/hashicorp/google": "2.5.5"
			}
		}
	}`)
}

func newDefaultProvider(name string) tfaddr.Provider {
	return tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", name)
}

func testConstraint(t *testing.T, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}
