package handlers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_formattingWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {}",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI)}, session.SessionNotInitialized.Err())
}

func TestLangServer_formatting_basic(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		StateStore:      ss,
		WalkerCollector: wc,
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
						Method:        "Format",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
							[]byte("provider  \"test\"   {\n\n}\n"),
						},
						ReturnArguments: []interface{}{
							[]byte("provider \"test\" {\n\n}\n"),
							nil,
						},
					},
				},
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
			"text": "provider  \"test\"   {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
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
						"start": { "line": 0, "character": 0 },
						"end": { "line": 1, "character": 0 }
					},
					"newText": "provider \"test\" {\n"
				}
			]
		}`)
}

func TestLangServer_formatting_oldVersion(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		StateStore:      ss,
		WalkerCollector: wc,
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
							version.Must(version.NewVersion("0.7.6")),
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
						Method:        "Format",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
							[]byte("provider  \"test\"   {\n\n}\n"),
						},
						ReturnArguments: []interface{}{
							nil,
							errors.New("not implemented"),
						},
					},
				},
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
			"text": "provider  \"test\"   {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			}
		}`, tmpDir.URI)}, code.SystemError.Err())
}

func TestLangServer_formatting_variables(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		StateStore:      ss,
		WalkerCollector: wc,
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
						Method:        "Format",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
							[]byte("test  = \"dev\""),
						},
						ReturnArguments: []interface{}{
							[]byte("test = \"dev\""),
							nil,
						},
					},
				},
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
			"languageId": "terraform-vars",
			"text": "test  = \"dev\"",
			"uri": "%s/terraform.tfvars"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/terraform.tfvars"
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": [
				{
					"range": {
						"start": { "line": 0, "character": 0 },
						"end": { "line": 0, "character": 13 }
					},
					"newText": "test = \"dev\""
				}
			]
		}`)
}

func TestLangServer_formatting_diffBug(t *testing.T) {
	tmpDir := TempDir(t)

	cfg := `resource "aws_lambda_function" "f" {
    environment {
        variables = {
            a = "b"
        }
    }
}
`
	formattedCfg := `resource "aws_lambda_function" "f" {
  environment {
    variables = {
      a = "b"
    }
  }
}
`

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		StateStore:      ss,
		WalkerCollector: wc,
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
						Method:        "Format",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType("*context.valueCtx"),
							[]byte(cfg),
						},
						ReturnArguments: []interface{}{
							[]byte(formattedCfg),
							nil,
						},
					},
				},
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
			"text": `+fmt.Sprintf("%q", cfg)+`,
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
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
							"line": 1,
							"character": 0
						},
						"end": {
							"line": 5,
							"character": 0
						}
					},
					"newText": "  environment {\n    variables = {\n      a = \"b\"\n"
				},
				{
					"range": {
						"start": {
							"line": 6,
							"character": 0
						},
						"end": {
							"line": 6,
							"character": 0
						}
					},
					"newText":"  }\n"
				}
			]
		}`)
}
