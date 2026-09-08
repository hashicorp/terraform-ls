// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package rootmodules

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TestRootModulesFeature_TerraformVersion(t *testing.T) {
	ss, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	fs := filesystem.NewFilesystem(ss.DocumentStore)

	type records struct {
		path    string
		version *version.Version
	}

	testCases := []struct {
		name    string
		records []records
		path    string
		version *version.Version
	}{
		{
			"no records",
			[]records{},
			"path/to/module",
			nil,
		},
		{
			"matching record exists",
			[]records{
				{
					"path/to/module",
					version.Must(version.NewVersion("0.12.0")),
				},
			},
			"path/to/module",
			version.Must(version.NewVersion("0.12.0")),
		},
		{
			"no exact match",
			[]records{
				{
					"path/to/module",
					version.Must(version.NewVersion("0.12.0")),
				},
			},
			"path/another/module",
			version.Must(version.NewVersion("0.12.0")),
		},
		{
			"no exact match, multiple records",
			[]records{
				{
					"path/to/module",
					nil,
				},
				{
					"path/another/module",
					nil,
				},
				{
					"root",
					version.Must(version.NewVersion("0.12.0")),
				},
			},
			"path/random/module",
			version.Must(version.NewVersion("0.12.0")),
		},
		{
			"exact match, multiple records",
			[]records{
				{
					"path/to/module",
					nil,
				},
				{
					"path/another/module",
					nil,
				},
				{
					"root",
					version.Must(version.NewVersion("0.12.0")),
				},
			},
			"path/another/module",
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			feature, err := NewRootModulesFeature(eventBus, ss, fs, exec.NewMockExecutor(nil))
			if err != nil {
				t.Fatal(err)
			}

			for _, record := range tc.records {
				feature.Store.Add(record.path)
				feature.Store.UpdateTerraformAndProviderVersions(record.path, record.version, nil, nil)
			}

			version := feature.TerraformVersion(tc.path)

			if diff := cmp.Diff(version, tc.version); diff != "" {
				t.Fatalf("version mismatch for %q: %s", tc.path, diff)
			}
		})
	}
}

func TestRootModulesFeature_DiscoverDidOpen(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	eventBus := eventbus.NewEventBus()
	fs := filesystem.NewFilesystem(gs.DocumentStore)

	feature, err := NewRootModulesFeature(eventBus, gs, fs, exec.NewMockExecutor(nil))
	if err != nil {
		t.Fatal(err)
	}

	rootPath := t.TempDir()
	ctx := lsctx.WithDocumentContext(context.Background(), lsctx.Document{})

	dir, err := feature.discover(rootPath, []string{".terraform.lock.hcl"})
	if err != nil {
		t.Fatal(err)
	}
	if dir == nil {
		t.Fatalf("expected discover to register root module %q", rootPath)
	}

	if _, err := feature.didOpen(ctx, *dir); err != nil {
		t.Fatal(err)
	}

	ids, err := gs.JobStore.ListIncompleteJobsForDir(*dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatalf("expected jobs for inited root module %q", rootPath)
	}
}
