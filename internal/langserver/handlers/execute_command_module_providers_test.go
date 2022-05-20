package handlers

import (
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_workspaceExecuteCommand_moduleProviders_argumentError(t *testing.T) {
	rootDir := t.TempDir()
	rootUri := uri.FromPath(rootDir)

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				rootDir: validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, rootUri)})
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
	}`, fmt.Sprintf("%s/main.tf", rootUri))})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q
	}`, cmd.Name("module.providers"))}, code.InvalidParams.Err())
}

func TestLangServer_workspaceExecuteCommand_moduleProviders_basic(t *testing.T) {
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
		ProviderRequirements: map[tfaddr.Provider]version.Constraints{
			tfaddr.NewDefaultProvider("aws"):    testConstraint(t, "1.2.3"),
			tfaddr.NewDefaultProvider("google"): testConstraint(t, ">= 2.0.0"),
		},
		ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
			{LocalName: "aws"}:    tfaddr.NewDefaultProvider("aws"),
			{LocalName: "google"}: tfaddr.NewDefaultProvider("google"),
		},
	}

	err = s.Modules.UpdateMetadata(modDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	pVersions := map[tfaddr.Provider]*version.Version{
		tfaddr.NewDefaultProvider("aws"):    version.Must(version.NewVersion("1.2.3")),
		tfaddr.NewDefaultProvider("google"): version.Must(version.NewVersion("2.5.5")),
	}
	err = s.Modules.UpdateInstalledProviders(modDir, pVersions)
	if err != nil {
		t.Fatal(err)
	}

	wc := module.NewWalkerCollector()

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

func testConstraint(t *testing.T, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}
