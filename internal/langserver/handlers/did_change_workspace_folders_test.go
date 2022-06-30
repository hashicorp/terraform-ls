package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func TestDidChangeWorkspaceFolders(t *testing.T) {
	rootDir := TempDir(t)

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
		"processId": 12345,
		"workspaceFolders": [
			{
				"uri": %q,
				"name": "first"
			}
		]
	}`, rootDir.URI, rootDir.URI)})
	waitForWalkerPath(t, ss, wc, rootDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWorkspaceFolders",
		ReqParams: fmt.Sprintf(`{
		"event": {
			"added": [
				{"uri": %q, "name": "second"}
			],
			"removed": [
				{"uri": %q, "name": "first"}
			]
		}
	}`, rootDir.URI, rootDir.URI)})
	waitForWalkerPath(t, ss, wc, rootDir)
}
