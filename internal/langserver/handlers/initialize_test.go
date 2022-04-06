package handlers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/stretchr/testify/mock"
)

func TestInitialize_twice(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

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
	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, tmpDir.URI)}, code.SystemError.Err())
}

func TestInitialize_withIncompatibleTerraformVersion(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

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
							version.Must(version.NewVersion("0.11.0")),
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
	    "processId": 12345,
	    "rootUri": %q
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
}

func TestInitialize_withInvalidRootURI(t *testing.T) {
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

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: `{
	    "capabilities": {},
	    "processId": 12345,
	    "rootUri": "meh"
	}`}, code.SystemError.Err())
}

func TestInitialize_multipleFolders(t *testing.T) {
	rootDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

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
	    		"name": "root"
	    	}
	    ]
	}`, rootDir.URI, rootDir.URI)})
	waitForWalkerPath(t, ss, wc, rootDir)
}

func TestInitialize_ignoreDirectoryNames(t *testing.T) {
	tmpDir := TempDir(t, "plugin", "ignore")
	pluginDir := filepath.Join(tmpDir.Path(), "plugin")
	emptyDir := filepath.Join(tmpDir.Path(), "ignore")

	InitPluginCache(t, pluginDir)
	InitPluginCache(t, emptyDir)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				pluginDir: validTfMockCalls(),
				emptyDir: {
					// TODO! improve mock and remove entry for `emptyDir` here afterwards
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
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
			"processId": 12345,
			"initializationOptions": {
				"ignoreDirectoryNames": [%q]
			}
	}`, tmpDir.URI, "ignore")})
	waitForWalkerPath(t, ss, wc, tmpDir)
}
