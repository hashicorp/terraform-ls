// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func TestInitialize_twice(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

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
	}`, tmpDir.URI)}, jrpc2.SystemError.Err())
}

func TestInitialize_withIncompatibleTerraformVersion(t *testing.T) {
	tmpDir := TempDir(t)

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
	}`}, jrpc2.InvalidParams.Err())
}

func TestInitialize_multipleFolders(t *testing.T) {
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
	wc := walker.NewWalkerCollector()

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
				"indexing": {
					"ignoreDirectoryNames": [%q]
				}
			}
	}`, tmpDir.URI, "ignore")})
	waitForWalkerPath(t, ss, wc, tmpDir)
}

func TestInitialize_differentWorkspaceLayouts(t *testing.T) {
	testData, err := filepath.Abs("testdata-initialize")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		root string

		expectedModules     []string
		expectedRootModules []string
	}{
		{
			filepath.Join(testData, "uninitialized-root"),
			[]string{
				filepath.Join(testData, "uninitialized-root"),
			},
			[]string{},
		},
		{
			filepath.Join(testData, "single-root-ext-modules-only"),
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "codelabs", "simple"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "ilb_routing"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "multi_vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "secondary_ranges"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "simple_project"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "submodule_firewall"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "submodule_network_peering"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "examples", "submodule_svpc_access"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "fabric-net-firewall"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "fabric-net-svpc-access"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "network-peering"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "routes-beta"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "subnets-beta"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "all_examples"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "ilb_routing"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "multi_vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "secondary_ranges"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_firewall"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_network_peering"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "test", "setup"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "codelabs", "simple"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "ilb_routing"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "multi_vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "secondary_ranges"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "simple_project"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "submodule_firewall"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "submodule_network_peering"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "examples", "submodule_svpc_access"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "fabric-net-firewall"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "fabric-net-svpc-access"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "network-peering"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "routes-beta"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "subnets-beta"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "all_examples"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "ilb_routing"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "multi_vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "secondary_ranges"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_firewall"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_network_peering"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "test", "setup"),
			},
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
			},
		},

		{
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "codelabs", "simple"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "ilb_routing"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "multi_vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "secondary_ranges"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "simple_project"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "submodule_firewall"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "submodule_network_peering"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "examples", "submodule_svpc_access"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "fabric-net-firewall"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "fabric-net-svpc-access"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "network-peering"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "routes-beta"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "subnets-beta"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "all_examples"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "ilb_routing"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "multi_vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "secondary_ranges"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_firewall"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_network_peering"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "test", "setup"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "codelabs", "simple"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "ilb_routing"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "multi_vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "secondary_ranges"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "simple_project"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "submodule_firewall"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "submodule_network_peering"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "examples", "submodule_svpc_access"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "fabric-net-firewall"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "fabric-net-svpc-access"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "network-peering"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "routes-beta"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "subnets-beta"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "all_examples"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "delete_default_gateway_routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "ilb_routing"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "multi_vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "secondary_ranges"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "simple_project_with_regional_network"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_firewall"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "fixtures", "submodule_network_peering"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "test", "setup"),
				filepath.Join(testData, "single-root-local-and-ext-modules", "alpha"),
				filepath.Join(testData, "single-root-local-and-ext-modules", "beta"),
				filepath.Join(testData, "single-root-local-and-ext-modules", "charlie"),
			},
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},

		{
			filepath.Join(testData, "single-root-local-modules-only"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
				filepath.Join(testData, "single-root-local-modules-only", "alpha"),
				filepath.Join(testData, "single-root-local-modules-only", "beta"),
				filepath.Join(testData, "single-root-local-modules-only", "charlie"),
			},
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},

		{
			filepath.Join(testData, "single-root-no-modules"),
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-ext-modules-only"),
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "alpha"),
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "beta"),
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "charlie"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module"),
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},

		// Multi-root

		{
			filepath.Join(testData, "main-module-multienv"),
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
				filepath.Join(testData, "main-module-multienv", "env", "staging"),
				filepath.Join(testData, "main-module-multienv", "main"),
				filepath.Join(testData, "main-module-multienv", "modules", "application"),
				filepath.Join(testData, "main-module-multienv", "modules", "database"),
			},
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
				filepath.Join(testData, "main-module-multienv", "env", "staging"),
			},
		},

		{
			filepath.Join(testData, "multi-root-no-modules"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
				filepath.Join(testData, "multi-root-no-modules", "third-root"),
			},
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
				filepath.Join(testData, "multi-root-no-modules", "third-root"),
			},
		},

		{
			filepath.Join(testData, "multi-root-local-modules-down"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "first-root", "alpha"),
				filepath.Join(testData, "multi-root-local-modules-down", "first-root", "beta"),
				filepath.Join(testData, "multi-root-local-modules-down", "first-root", "charlie"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root", "alpha"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root", "beta"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root", "charlie"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root", "alpha"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root", "beta"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root", "charlie"),
			},
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root"),
			},
		},

		{
			filepath.Join(testData, "multi-root-local-modules-up"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-up", "main-module"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "first"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "second"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "third"),
			},
			[]string{
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "first"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "second"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "third"),
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.root), func(t *testing.T) {
			ctx := context.Background()
			dir := document.DirHandleFromPath(tc.root)

			ss, err := state.NewStateStore()
			if err != nil {
				t.Fatal(err)
			}
			eventBus := eventbus.NewEventBus()
			mockCalls := &exec.TerraformMockCalls{
				PerWorkDir: map[string][]*mock.Call{
					dir.Path(): validTfMockCalls(),
				},
			}
			fs := filesystem.NewFilesystem(ss.DocumentStore)
			features, err := NewTestFeatures(eventBus, ss, fs, mockCalls)
			if err != nil {
				t.Fatal(err)
			}
			features.Modules.Start(ctx)
			defer features.Modules.Stop()
			features.RootModules.Start(ctx)
			defer features.RootModules.Stop()
			features.Variables.Start(ctx)
			defer features.Variables.Stop()
			features.Stacks.Start(ctx)
			defer features.Stacks.Stop()
			features.Tests.Start(ctx)
			defer features.Tests.Stop()

			wc := walker.NewWalkerCollector()

			ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
				TerraformCalls:  mockCalls,
				StateStore:      ss,
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
				"processId": 12345,
				"workspaceFolders": [
					{
						"uri": %q,
						"name": "root"
					}
				]
			}`, dir.URI, dir.URI)})
			waitForWalkerPath(t, ss, wc, dir)
			ls.Notify(t, &langserver.CallRequest{
				Method:    "initialized",
				ReqParams: "{}",
			})

			// Verify the number of modules
			allModules, err := features.Modules.Store.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(allModules) != len(tc.expectedModules) {
				for _, mods := range tc.expectedModules {
					t.Logf("expected module: %s", mods)
				}
				for _, mods := range allModules {
					t.Logf("got module: %s", mods.Path())
				}
				t.Fatalf("expected %d modules, got %d", len(tc.expectedModules), len(allModules))
			}
			for _, path := range tc.expectedModules {
				_, err := features.Modules.Store.ModuleRecordByPath(path)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Verify the number of root modules
			allRootModules, err := features.RootModules.Store.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(allRootModules) != len(tc.expectedRootModules) {
				for _, mods := range tc.expectedRootModules {
					t.Logf("expected root module: %s", mods)
				}
				for _, mods := range allRootModules {
					t.Logf("got root module: %s", mods.Path())
				}
				t.Fatalf("expected %d root modules, got %d", len(tc.expectedRootModules), len(allRootModules))
			}
			for _, path := range tc.expectedRootModules {
				_, err := features.RootModules.Store.RootRecordByPath(path)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
