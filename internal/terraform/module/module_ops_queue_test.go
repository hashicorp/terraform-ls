package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestModuleOpsQueue_modulePriority(t *testing.T) {
	mq := newModuleOpsQueue()

	fs := filesystem.NewFilesystem()
	fs.SetLogger(testLogger())

	dir := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	ops := []ModuleOperation{
		NewModuleOperation(
			closedModAtPath(t, fs, dir, "alpha"),
			OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			openModAtPath(t, fs, dir, "beta"),
			OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			openModAtPath(t, fs, dir, "gamma"),
			OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			closedModAtPath(t, fs, dir, "delta"),
			OpTypeGetTerraformVersion,
		),
	}

	for _, op := range ops {
		mq.PushOp(op)
	}

	firstOp, _ := mq.PopOp()

	expectedFirstPath := filepath.Join(dir, "beta")
	firstPath := firstOp.Module.Path()
	if firstPath != expectedFirstPath {
		t.Fatalf("path mismatch\nexpected: %s\ngiven:    %s",
			expectedFirstPath, firstPath)
	}

	secondOp, _ := mq.PopOp()
	expectedSecondPath := filepath.Join(dir, "gamma")
	secondPath := secondOp.Module.Path()
	if secondPath != expectedSecondPath {
		t.Fatalf("path mismatch\nexpected: %s\ngiven:    %s",
			expectedSecondPath, secondPath)
	}
}

func closedModAtPath(t *testing.T, fs filesystem.Filesystem, dir, modName string) Module {
	modPath := filepath.Join(dir, modName)

	docPath := filepath.Join(modPath, "main.tf")
	dh := ilsp.FileHandlerFromDocumentURI(protocol.DocumentURI(uri.FromPath(docPath)))
	err := fs.CreateDocument(dh, []byte{})
	if err != nil {
		t.Fatal(err)
	}
	m := newModule(fs, modPath)
	m.SetLogger(testLogger())
	return m
}

func openModAtPath(t *testing.T, fs filesystem.Filesystem, dir, modName string) Module {
	modPath := filepath.Join(dir, modName)
	docPath := filepath.Join(modPath, "main.tf")
	dh := ilsp.FileHandlerFromDocumentURI(protocol.DocumentURI(uri.FromPath(docPath)))
	err := fs.CreateAndOpenDocument(dh, []byte{})
	if err != nil {
		t.Fatal(err)
	}
	m := newModule(fs, modPath)
	m.SetLogger(testLogger())
	return m
}
