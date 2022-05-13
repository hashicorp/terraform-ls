package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_DidChangeWatchedFiles_file(t *testing.T) {
	tmpDir := TempDir(t)

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
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	// Verify main.tf was parsed
	mod, err := ss.Modules.ModuleByPath(tmpDir.Path())
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
	newSrc := `variable "original" {
  default = "foo"
}
`
	err = os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(newSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Verify nothing has changed yet
	mod, err = ss.Modules.ModuleByPath(tmpDir.Path())
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

	// Verify file was re-parsed
	mod, err = ss.Modules.ModuleByPath(tmpDir.Path())
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

func TestLangServer_DidChangeWatchedFiles_dir(t *testing.T) {
	tmpDir := TempDir(t)

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
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	// Verify main.tf was parsed
	mod, err := ss.Modules.ModuleByPath(tmpDir.Path())
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
	newSrc := `variable "original" {
  default = "foo"
}
`
	err = os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte(newSrc), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Verify nothing has changed yet
	mod, err = ss.Modules.ModuleByPath(tmpDir.Path())
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

	// Verify file was re-parsed
	mod, err = ss.Modules.ModuleByPath(tmpDir.Path())
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
