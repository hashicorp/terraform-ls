// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package walker

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/indexer"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/scheduler"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestWalker_basic(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	pa := state.NewPathAwaiter(ss.WalkerPaths, false)

	walkFunc := func(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
		return job.IDs{}, nil
	}

	w := NewWalker(fs, pa, ss.Modules, walkFunc)
	w.Collector = NewWalkerCollector()
	w.SetLogger(testLogger())

	root, err := filepath.Abs(filepath.Join("testdata", "uninitialized-root"))
	if err != nil {
		t.Fatal(err)
	}
	dir := document.DirHandleFromPath(root)

	ctx := context.Background()
	err = ss.WalkerPaths.EnqueueDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = w.StartWalking(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = ss.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{dir})
	if err != nil {
		t.Fatal(err)
	}
	err = ss.JobStore.WaitForJobs(ctx, w.Collector.JobIds()...)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Collector.ErrorOrNil()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWalker_complexModules(t *testing.T) {
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		root             string
		totalModuleCount int

		expectedModules     []string
		expectedSchemaPaths []string
	}{
		{
			filepath.Join(testData, "single-root-ext-modules-only"),
			1,
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
			1,
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
			1,
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
			1,
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-ext-modules-only"),
			1,
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			1,
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
			1,
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
			3,
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
			3,
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
			3,
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
			3,
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

	ctx := context.Background()

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.root), func(t *testing.T) {
			ss, err := state.NewStateStore()
			if err != nil {
				t.Fatal(err)
			}
			ss.SetLogger(testLogger())

			fs := filesystem.NewFilesystem(ss.DocumentStore)
			tfCalls := &exec.TerraformMockCalls{
				AnyWorkDir: validTfMockCalls(tc.totalModuleCount),
			}
			ctx = exec.WithExecutorOpts(ctx, &exec.ExecutorOpts{
				ExecPath: "tf-mock",
			})

			s := scheduler.NewScheduler(ss.JobStore, 1, job.LowPriority)
			ss.SetLogger(testLogger())
			s.Start(ctx)

			pa := state.NewPathAwaiter(ss.WalkerPaths, false)
			indexer := indexer.NewIndexer(fs, ss.Modules, ss.Vars, ss.ProviderSchemas, ss.RegistryModules, ss.JobStore,
				exec.NewMockExecutor(tfCalls), registry.NewClient())
			indexer.SetLogger(testLogger())
			w := NewWalker(fs, pa, ss.Modules, ss.Vars, indexer.WalkedModule)
			w.Collector = NewWalkerCollector()
			w.SetLogger(testLogger())
			dir := document.DirHandleFromPath(tc.root)
			err = ss.WalkerPaths.EnqueueDir(ctx, dir)
			if err != nil {
				t.Fatal(err)
			}
			ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
			err = w.StartWalking(ctx)
			if err != nil {
				t.Fatal(err)
			}
			err = ss.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{dir})
			if err != nil {
				t.Fatal(err)
			}
			err = ss.JobStore.WaitForJobs(ctx, w.Collector.JobIds()...)
			if err != nil {
				t.Fatal(err)
			}
			err = w.Collector.ErrorOrNil()
			if err != nil {
				t.Fatal(err)
			}

			s.Stop()

			modules, err := ss.Modules.List()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedModules, modulePaths(modules)); diff != "" {
				t.Fatalf("modules don't match: %s", diff)
			}

			it, err := ss.ProviderSchemas.ListSchemas()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedSchemaPaths, localProviderSchemaPaths(t, it)); diff != "" {
				t.Fatalf("schemas don't match: %s", diff)
			}
		})
	}
}

func modulePaths(modules []*state.Module) []string {
	paths := make([]string, len(modules))

	for i, mod := range modules {
		paths[i] = mod.Path
	}

	return paths
}

func localProviderSchemaPaths(t *testing.T, it *state.ProviderSchemaIterator) []string {
	schemas := make([]string, 0)

	for ps := it.Next(); ps != nil; ps = it.Next() {
		localSrc, ok := ps.Source.(state.LocalSchemaSource)
		if !ok {
			t.Fatalf("expected only local sources, found: %q", ps.Source)
		}

		schemas = append(schemas, localSrc.ModulePath)
	}

	return schemas
}

func validTfMockCalls(repeatability int) []*mock.Call {
	return []*mock.Call{
		{
			Method: "Version",
			// Repeatability: repeatability,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.valueCtx"),
			},
			ReturnArguments: []interface{}{
				version.Must(version.NewVersion("0.12.0")),
				nil,
				nil,
			},
		},
		{
			Method: "GetExecPath",
			// Repeatability: repeatability,
			ReturnArguments: []interface{}{
				"",
			},
		},
		{
			Method: "ProviderSchemas",
			// Repeatability: repeatability,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.valueCtx"),
			},
			ReturnArguments: []interface{}{
				testProviderSchema,
				nil,
			},
		},
	}
}

var testProviderSchema = &tfjson.ProviderSchemas{
	FormatVersion: "0.1",
	Schemas: map[string]*tfjson.ProviderSchema{
		"test": {
			ConfigSchema: &tfjson.Schema{},
		},
	},
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}
