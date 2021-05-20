package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/uri"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_workspaceExecuteCommand_moduleCallers_argumentError(t *testing.T) {
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
	}`, cmd.Name("module.callers"))}, code.InvalidParams.Err())
}

func TestLangServer_workspaceExecuteCommand_moduleCallers_basic(t *testing.T) {
	rootDir := t.TempDir()
	rootUri := uri.FromPath(rootDir)
	baseDirUri := uri.FromPath(filepath.Join(rootDir, "base"))

	createModuleCalling(t, "../base", filepath.Join(rootDir, "dev"))
	createModuleCalling(t, "../base", filepath.Join(rootDir, "staging"))
	createModuleCalling(t, "../base", filepath.Join(rootDir, "prod"))

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
	}`, fmt.Sprintf("%s/main.tf", baseDirUri))})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q,
		"arguments": ["uri=%s"]
	}`, cmd.Name("module.callers"), baseDirUri)}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 3,
		"result": {
			"v": 0,
			"callers": [
				{
					"uri": "%s/dev"
				},
				{
					"uri": "%s/prod"
				},
				{
					"uri": "%s/staging"
				}
			]
		}
	}`, rootUri, rootUri, rootUri))
}

func createModuleCalling(t *testing.T, src, modPath string) {
	modulesDir := filepath.Join(modPath, ".terraform", "modules")
	err := os.MkdirAll(modulesDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	configBytes := []byte(fmt.Sprintf(`
module "local" {
  source = %q
}
`, src))
	err = os.WriteFile(filepath.Join(modPath, "module.tf"), configBytes, 0755)
	if err != nil {
		t.Fatal(err)
	}

	manifestBytes := []byte(fmt.Sprintf(`{
    "Modules": [
        {
            "Key": "",
            "Source": "",
            "Dir": "."
        },
        {
            "Key": "local",
            "Source": %q,
            "Dir": %q
        }
    ]
}`, src, src))
	err = os.WriteFile(filepath.Join(modulesDir, "modules.json"), manifestBytes, 0755)
	if err != nil {
		t.Fatal(err)
	}
}
