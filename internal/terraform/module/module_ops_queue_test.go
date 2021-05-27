package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestModuleOpsQueue_modulePriority(t *testing.T) {
	fs := filesystem.NewFilesystem()
	fs.SetLogger(testLogger())

	mq := newModuleOpsQueue(fs)

	dir := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	ops := []ModuleOperation{
		NewModuleOperation(
			closedModPath(t, fs, dir, "alpha"),
			op.OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			openModAtPath(t, fs, dir, "beta"),
			op.OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			openModAtPath(t, fs, dir, "gamma"),
			op.OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			closedModPath(t, fs, dir, "delta"),
			op.OpTypeGetTerraformVersion,
		),
	}

	for _, op := range ops {
		mq.PushOp(op)
	}

	firstOp, ok := mq.PopOp()
	if !ok {
		t.Fatal("expected PopOp to succeed")
	}

	expectedFirstPath := filepath.Join(dir, "beta")
	firstPath := firstOp.ModulePath
	if firstPath != expectedFirstPath {
		t.Fatalf("path mismatch (1)\nexpected: %s\ngiven:    %s",
			expectedFirstPath, firstPath)
	}

	secondOp, _ := mq.PopOp()
	expectedSecondPath := filepath.Join(dir, "gamma")
	secondPath := secondOp.ModulePath
	if secondPath != expectedSecondPath {
		t.Fatalf("path mismatch (2)\nexpected: %s\ngiven:    %s",
			expectedSecondPath, secondPath)
	}
}

func closedModPath(t *testing.T, fs filesystem.Filesystem, dir, modName string) string {
	modPath := filepath.Join(dir, modName)

	docPath := filepath.Join(modPath, "main.tf")
	dh := ilsp.FileHandlerFromDocumentURI(protocol.DocumentURI(uri.FromPath(docPath)))
	err := fs.CreateDocument(dh, "test", []byte{})
	if err != nil {
		t.Fatal(err)
	}

	return modPath
}

func openModAtPath(t *testing.T, fs filesystem.Filesystem, dir, modName string) string {
	modPath := filepath.Join(dir, modName)
	docPath := filepath.Join(modPath, "main.tf")
	dh := ilsp.FileHandlerFromDocumentURI(protocol.DocumentURI(uri.FromPath(docPath)))
	err := fs.CreateAndOpenDocument(dh, "test", []byte{})
	if err != nil {
		t.Fatal(err)
	}

	return modPath
}
