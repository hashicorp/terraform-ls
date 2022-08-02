package handlers

import (
	"encoding/json"
	"fmt"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
)

func TestCompletionResolve_withoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method:    "completionItem/resolve",
		ReqParams: "{}"}, session.SessionNotInitialized.Err())
}

func TestCompletionResolve_withoutHook(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "completionItem/resolve",
		ReqParams: fmt.Sprintf(`{
			"label": "\"test\"",
			"kind": 1,
			"data": {
				"resolve_hook": "test",
				"path": "%s/main.tf"
			}
		}`, TempDir(t).URI),
	}, fmt.Sprintf(`{
			"jsonrpc": "2.0",
			"id": 2,
			"result": {
				"label": "\"test\"",
				"labelDetails": {},
				"kind": 1,
				"data": {
					"resolve_hook": "test",
					"path": "%s/main.tf"
				}
			}
	}`, TempDir(t).URI))
}
