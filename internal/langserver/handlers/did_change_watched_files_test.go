// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_DidChangeWatchedFiles_change_file(t *testing.T) {
	tmpDir := TempDir(t)
	ctx := context.Background()

	InitPluginCache(t, tmpDir.Path())

	originalSrc := `variable "original" {
  default = "foo"
}
`
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(originalSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			tmpDir.Path(): validTfMockCalls(),
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "",
			"uri": "%s/another.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was parsed
	mod, err := features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles := mod.ParsedModuleFiles.AsMap()
	parsedFile, ok := parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	// Change main.tf on disk
	newSrc := `variable "new" {
  default = "foo"
}
`
	err = os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(newSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Verify nothing has changed yet
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": "%s/main.tf",
            "type": 2
        }
    ]
}`, TempDir(t).URI)})
	waitForAllJobs(t, ss)

	// Verify file was re-parsed
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(newSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}
}

func TestLangServer_DidChangeWatchedFiles_create_file(t *testing.T) {
	tmpDir := TempDir(t)
	ctx := context.Background()

	InitPluginCache(t, tmpDir.Path())

	originalSrc := `variable "original" {
  default = "foo"
}
`
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(originalSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			tmpDir.Path(): {
				{
					Method:        "Version",
					Repeatability: 2,
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
					Method:        "ProviderSchemas",
					Repeatability: 2,
					Arguments: []interface{}{
						mock.AnythingOfType(""),
					},
					ReturnArguments: []interface{}{
						&tfjson.ProviderSchemas{
							FormatVersion: "0.1",
							Schemas: map[string]*tfjson.ProviderSchema{
								"test": {
									ConfigSchema: &tfjson.Schema{},
								},
							},
						},
						nil,
					},
				},
			},
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "variable \"original\" {\n  default = \"foo\"\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was parsed
	mod, err := features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles := mod.ParsedModuleFiles.AsMap()
	parsedFile, ok := parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	// Create another.tf on disk
	newSrc := `variable "another" {
  default = "foo"
}
`
	err = os.WriteFile(filepath.Join(tmpDir.Path(), "another.tf"), []byte(newSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Verify another.tf was not parsed *yet*
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	_, ok = parsedFiles["another.tf"]
	if ok {
		t.Fatalf("not expected to be parsed: %q", "another.tf")
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": "%s/main.tf",
            "type": 1
        }
    ]
}`, TempDir(t).URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	waitForAllJobs(t, ss)

	// Verify another.tf was parsed
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["another.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "another.tf")
	}
	if diff := cmp.Diff(newSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "another.tf", diff)
	}
}

func TestLangServer_DidChangeWatchedFiles_delete_file(t *testing.T) {
	tmpDir := TempDir(t)
	ctx := context.Background()

	InitPluginCache(t, tmpDir.Path())

	originalSrc := `variable "original" {
  default = "foo"
}
`
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(originalSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			tmpDir.Path(): validTfMockCalls(),
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "",
			"uri": "%s/another.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was parsed
	mod, err := features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles := mod.ParsedModuleFiles.AsMap()
	parsedFile, ok := parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	// Delete main.tf from disk
	err = os.Remove(filepath.Join(tmpDir.Path(), "main.tf"))
	if err != nil {
		t.Fatal(err)
	}

	// Verify main.tf still remains parsed
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": "%s/main.tf",
            "type": 3
        }
    ]
}`, TempDir(t).URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was deleted
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	_, ok = parsedFiles["main.tf"]
	if ok {
		t.Fatalf("not expected file to be parsed: %q", "main.tf")
	}
}

func TestLangServer_DidChangeWatchedFiles_change_dir(t *testing.T) {
	tmpDir := TempDir(t)
	ctx := context.Background()

	InitPluginCache(t, tmpDir.Path())

	originalSrc := `variable "original" {
  default = "foo"
}
`
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(originalSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			tmpDir.Path(): validTfMockCalls(),
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "",
			"uri": "%s/another.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was parsed
	mod, err := features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles := mod.ParsedModuleFiles.AsMap()
	parsedFile, ok := parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	// Change main.tf on disk
	newSrc := `variable "new" {
  default = "foo"
}
`
	err = os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(newSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Verify nothing has changed yet
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": %q,
            "type": 2
        }
    ]
}`, TempDir(t).URI)})
	waitForAllJobs(t, ss)

	// Verify file was re-parsed
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(newSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}
}

func TestLangServer_DidChangeWatchedFiles_create_dir(t *testing.T) {
	tmpDir := TempDir(t)
	ctx := context.Background()

	InitPluginCache(t, tmpDir.Path())

	originalSrc := `variable "original" {
  default = "foo"
}
`
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(originalSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			tmpDir.Path(): validTfMockCalls(),
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "variable \"original\" {\n  default = \"foo\"\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was parsed
	mod, err := features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles := mod.ParsedModuleFiles.AsMap()
	parsedFile, ok := parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	// Create new ./submodule w/ main.tf on disk
	submodPath := filepath.Join(tmpDir.Path(), "submodule")
	submodHandle := document.DirHandleFromPath(submodPath)
	err = os.Mkdir(submodPath, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	newSrc := `variable "new" {
  default = "foo"
}
`
	err = os.WriteFile(filepath.Join(submodPath, "main.tf"), []byte(newSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	InitPluginCache(t, submodHandle.Path())

	// Verify submodule was not parsed yet
	_, err = features.Modules.Store.ModuleRecordByPath(submodPath)
	if err == nil {
		t.Fatalf("%q: expected module not to be found", submodPath)
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": %q,
            "type": 1
        }
    ]
}`, submodHandle.URI)})
	waitForWalkerPath(t, ss, wc, submodHandle)
	waitForAllJobs(t, ss)

	// Verify submodule was discovered, but not parsed yet
	mod, err = features.Modules.Store.ModuleRecordByPath(submodPath)
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	_, ok = parsedFiles["main.tf"]
	if ok {
		t.Fatalf("file parsed: %q", "main.tf")
	}
}

func TestLangServer_DidChangeWatchedFiles_delete_dir(t *testing.T) {
	tmpDir := TempDir(t)
	ctx := context.Background()

	InitPluginCache(t, tmpDir.Path())

	originalSrc := `variable "original" {
  default = "foo"
}
`
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(originalSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			tmpDir.Path(): validTfMockCalls(),
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "variable \"original\" {\n  default = \"foo\"\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// Verify main.tf was parsed
	mod, err := features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles := mod.ParsedModuleFiles.AsMap()
	parsedFile, ok := parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	// Delete directory from disk
	err = os.RemoveAll(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}

	// Verify nothing has changed yet
	mod, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	parsedFiles = mod.ParsedModuleFiles.AsMap()
	parsedFile, ok = parsedFiles["main.tf"]
	if !ok {
		t.Fatalf("file not parsed: %q", "main.tf")
	}
	if diff := cmp.Diff(originalSrc, string(parsedFile.Bytes)); diff != "" {
		t.Fatalf("bytes mismatch for %q: %s", "main.tf", diff)
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": %q,
            "type": 3
        }
    ]
}`, TempDir(t).URI)})
	waitForAllJobs(t, ss)

	// Verify module is gone
	_, err = features.Modules.Store.ModuleRecordByPath(tmpDir.Path())
	if err == nil {
		t.Fatalf("expected module at %q to be gone", tmpDir.Path())
	}
}

func TestLangServer_DidChangeWatchedFiles_pluginChange(t *testing.T) {
	ctx := context.Background()
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	originalTestDir := filepath.Join(testData, "single-fake-provider")
	testDir := t.TempDir()
	// Copy test configuration so the test can run in isolation
	err = copy.Copy(originalTestDir, testDir)
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.DirHandleFromPath(testDir)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			testHandle.Path(): {
				{
					Method:        "Version",
					Repeatability: 2,
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
					Method:        "ProviderSchemas",
					Repeatability: 1,
					Arguments: []interface{}{
						mock.AnythingOfType(""),
					},
					ReturnArguments: []interface{}{
						&tfjson.ProviderSchemas{
							FormatVersion: "0.1",
							Schemas: map[string]*tfjson.ProviderSchema{
								"foo": {
									ConfigSchema: &tfjson.Schema{},
								},
							},
						},
						nil,
					},
				},
			},
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, testHandle.URI)})
	waitForWalkerPath(t, ss, wc, testHandle)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
				"textDocument": {
					"version": 0,
					"languageId": "terraform",
					"text": "provider \"foo\" {\n\n}\n",
					"uri": "%s/main.tf"
				}
			}`, testHandle.URI)})
	waitForAllJobs(t, ss)

	addr := tfaddr.MustParseProviderSource("-/foo")
	vc := version.MustConstraints(version.NewConstraint(">= 1.0"))

	_, err = ss.ProviderSchemas.ProviderSchema(testHandle.Path(), addr, vc)
	if err == nil {
		t.Fatal("expected -/foo schema to be missing")
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": "%s/.terraform.lock.hcl",
            "type": 1
        }
    ]
}`, testHandle.URI)})
	waitForAllJobs(t, ss)

	_, err = ss.ProviderSchemas.ProviderSchema(testHandle.Path(), addr, vc)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLangServer_DidChangeWatchedFiles_moduleInstalled(t *testing.T) {
	ctx := context.Background()
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	originalTestDir := filepath.Join(testData, "uninitialized-single-submodule")
	testDir := t.TempDir()
	// Copy test configuration so the test can run in isolation
	err = copy.Copy(originalTestDir, testDir)
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.DirHandleFromPath(testDir)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	eventBus := eventbus.NewEventBus()
	mockCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			testHandle.Path(): validTfMockCalls(),
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
	features.Search.Start(ctx)
	defer features.Search.Stop()

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
	    "processId": 12345
	}`, testHandle.URI)})
	waitForWalkerPath(t, ss, wc, testHandle)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	// Open a file of the module
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "module {\n  source = \"github.com/hashicorp/terraform-azurerm-hcp-consul?ref=v0.2.4\"\n}\n",
			"uri": "%s/main.tf"
		}
	}`, testHandle.URI)})
	waitForAllJobs(t, ss)

	submodulePath := filepath.Join(testDir, ".terraform", "modules", "azure-hcp-consul")
	submoduleHandle := document.DirHandleFromPath(submodulePath)
	_, err = features.Modules.Store.ModuleRecordByPath(submodulePath)
	if err == nil || !state.IsRecordNotFound(err) {
		t.Fatalf("expected submodule not to be found: %s", err)
	}

	// Install Terraform
	tfVersion := version.Must(version.NewVersion("1.1.7"))
	i := install.NewInstaller()
	execPath, err := i.Install(ctx, []src.Installable{
		&releases.ExactVersion{
			Product: product.Terraform,
			Version: tfVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Install submodule
	tf, err := exec.NewExecutor(testHandle.Path(), execPath)
	if err != nil {
		t.Fatal(err)
	}
	err = tf.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ls.Call(t, &langserver.CallRequest{
		Method: "workspace/didChangeWatchedFiles",
		ReqParams: fmt.Sprintf(`{
    "changes": [
        {
            "uri": "%s/.terraform/modules/modules.json",
            "type": 1
        }
    ]
}`, testHandle.URI)})
	waitForAllJobs(t, ss)
	waitForWalkerPath(t, ss, wc, submoduleHandle)

	// Verify submodule was indexed
	_, err = features.Modules.Store.ModuleRecordByPath(submodulePath)
	if err != nil {
		t.Fatal(err)
	}
}
