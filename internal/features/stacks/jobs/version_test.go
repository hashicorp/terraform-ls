// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func TestLoadTerraformVersion(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	stacksVersionFilePath := filepath.Join(testData, "stacks-version-file")

	err = ss.Add(stacksVersionFilePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = LoadTerraformVersion(ctx, testFs, ss, stacksVersionFilePath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := ss.StackRecordByPath(stacksVersionFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if record.RequiredTerraformVersion.String() != "1.9.0" {
		t.Fatalf("expected version 1.9.0, got %s", record.RequiredTerraformVersion.String())
	}

	if record.RequiredTerraformVersionState != operation.OpStateLoaded {
		t.Fatalf("expected state %s, got %s", operation.OpStateLoaded, record.RequiredTerraformVersionState)
	}

	if record.RequiredTerraformVersionErr != nil {
		t.Fatalf("expected nil error, got %s", record.RequiredTerraformVersionErr)
	}
}

func TestLoadTerraformVersion_invalid(t *testing.T) {

	testCases := []struct {
		testName string
		testDir  string
	}{
		{
			"invalid-version",
			"stacks-version-file-invalid",
		},
		{
			"no-version-file",
			"stacks-version-file-none",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.testName), func(t *testing.T) {
			ctx := context.Background()
			gs, err := globalState.NewStateStore()
			if err != nil {
				t.Fatal(err)
			}
			ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
			if err != nil {
				t.Fatal(err)
			}

			testData, err := filepath.Abs("testdata")
			if err != nil {
				t.Fatal(err)
			}
			testFs := filesystem.NewFilesystem(gs.DocumentStore)

			stacksVersionFilePath := filepath.Join(testData, tc.testDir)

			err = ss.Add(stacksVersionFilePath)
			if err != nil {
				t.Fatal(err)
			}

			ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
			err = LoadTerraformVersion(ctx, testFs, ss, stacksVersionFilePath)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			record, err := ss.StackRecordByPath(stacksVersionFilePath)
			if err != nil {
				t.Fatal(err)
			}

			if record.RequiredTerraformVersion != nil {
				t.Fatalf("expected nil version, got %s", record.RequiredTerraformVersion.String())
			}

			if record.RequiredTerraformVersionState != operation.OpStateLoaded {
				t.Fatalf("expected state %s, got %s", operation.OpStateLoaded, record.RequiredTerraformVersionState)
			}

			if record.RequiredTerraformVersionErr == nil {
				t.Fatal("expected error in record, got nil")
			}
		})
	}
}
