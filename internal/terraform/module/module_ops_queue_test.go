package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/state"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func TestModuleOpsQueue_modulePriority(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(testLogger())

	mq := newModuleOpsQueue(ss.DocumentStore)

	dir := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	ops := []ModuleOperation{
		NewModuleOperation(
			closedModPath(t, dir, "alpha"),
			op.OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			openModAtPath(t, ss.DocumentStore, dir, "beta"),
			op.OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			openModAtPath(t, ss.DocumentStore, dir, "gamma"),
			op.OpTypeGetTerraformVersion,
		),
		NewModuleOperation(
			closedModPath(t, dir, "delta"),
			op.OpTypeGetTerraformVersion,
		),
	}
	t.Logf("total operations: %d", len(ops))

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

func closedModPath(t *testing.T, dir, modName string) string {
	modPath := filepath.Join(dir, modName)

	err := os.Mkdir(modPath, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(modPath, "main.tf")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	return modPath
}

func openModAtPath(t *testing.T, ds *state.DocumentStore, dir, modName string) string {
	modPath := filepath.Join(dir, modName)

	dh := document.HandleFromPath(filepath.Join(modPath, "main.tf"))

	err := ds.OpenDocument(dh, "test", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	return modPath
}
