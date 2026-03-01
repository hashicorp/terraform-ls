package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfpolicytest "github.com/hashicorp/terraform-schema/policytest"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(PolicyTestRecord{}),
	cmp.Comparer(func(x, y version.Constraints) bool {
		return x.String() == y.String()
	}),
	cmp.Comparer(func(x, y hcl.File) bool {
		return (x.Body == y.Body && cmp.Equal(x.Bytes, y.Bytes))
	}),
	ctydebug.CmpOptions,
}

func setupStore(t *testing.T) (*PolicyTestStore, *globalState.StateStore) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewPolicyTestStore(gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}
	return s, gs
}

func TestPolicyStore_Add_duplicate(t *testing.T) {
	s, _ := setupStore(t)
	path := t.TempDir()
	_ = s.Add(path)

	err := s.Add(path)
	if err == nil {
		t.Fatal("expected error for duplicate entry")
	}
	var existsErr *globalState.AlreadyExistsError
	if !errors.As(err, &existsErr) {
		t.Fatalf("unexpected error type: %T", err)
	}
}

func TestPolicyStore_PolicyRecordByPath(t *testing.T) {
	s, _ := setupStore(t)
	path := t.TempDir()
	_ = s.Add(path)

	record, err := s.PolicyTestRecordByPath(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := &PolicyTestRecord{
		path:            path,
		MetaState:       op.OpStateUnknown,
		RefOriginsState: op.OpStateUnknown,
		RefTargetsState: op.OpStateUnknown,
		PolicyTestDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          op.OpStateUnknown,
			globalAst.SchemaValidationSource:    op.OpStateUnknown,
			globalAst.ReferenceValidationSource: op.OpStateUnknown,
			globalAst.TerraformValidateSource:   op.OpStateUnknown,
		},
	}

	if diff := cmp.Diff(expected, record, cmpOpts); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPolicyStore_UpdateMetadata(t *testing.T) {
	s, _ := setupStore(t)
	path := t.TempDir()
	_ = s.Add(path)

	meta := &tfpolicytest.Meta{
		Path:      path,
		Filenames: []string{"config.policytest.hcl"},
	}

	err := s.UpdateMetadata(path, meta, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, _ := s.PolicyTestRecordByPath(path)
	if record.MetaState != op.OpStateLoaded {
		t.Errorf("expected state Loaded, got %v", record.MetaState)
	}
}

func TestPolicyStore_UpdateParsedPolicyFiles(t *testing.T) {
	s, _ := setupStore(t)
	path := t.TempDir()
	_ = s.Add(path)

	p := hclparse.NewParser()
	f, _ := p.ParseHCL([]byte(`policy { consumer = terraform }`), "config.policy.hcl")

	files := ast.PolicyTestFiles{
		ast.PolicyTestFilename("config.policytest.hcl"): f,
	}

	err := s.UpdateParsedPolicyTestFiles(path, files, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, _ := s.PolicyTestRecordByPath(path)
	if diff := cmp.Diff(files, record.ParsedPolicyTestFiles, cmpOpts); diff != "" {
		t.Fatalf("parsed files mismatch (-want +got):\n%s", diff)
	}
}

func TestPolicyStore_UpdatePolicyDiagnostics(t *testing.T) {
	s, _ := setupStore(t)
	path := t.TempDir()
	_ = s.Add(path)

	diags := ast.PolicyTestDiags{
		ast.PolicyTestFilename("config.policytest.hcl"): {
			{Severity: hcl.DiagError, Summary: "Invalid resource policy"},
		},
	}

	err := s.UpdatePolicyTestDiagnostics(path, globalAst.SchemaValidationSource, diags)
	if err != nil {
		t.Fatal(err)
	}

	record, _ := s.PolicyTestRecordByPath(path)
	if record.PolicyTestDiagnosticsState[globalAst.SchemaValidationSource] != op.OpStateLoaded {
		t.Fatal("expected diagnostic state to be Loaded")
	}

	savedDiags := record.PolicyTestDiagnostics[globalAst.SchemaValidationSource]
	if savedDiags.Count() != 1 {
		t.Errorf("expected 1 diagnostic, got %d", savedDiags.Count())
	}
}

func TestPolicyStore_List(t *testing.T) {
	s, _ := setupStore(t)
	tmpDir := t.TempDir()

	paths := []string{
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
	}

	for _, p := range paths {
		_ = s.Add(p)
	}

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	expectedRecords := []*PolicyTestRecord{
		{
			path:            paths[0],
			MetaState:       op.OpStateUnknown,
			RefOriginsState: op.OpStateUnknown,
			RefTargetsState: op.OpStateUnknown,
			PolicyTestDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          op.OpStateUnknown,
				globalAst.SchemaValidationSource:    op.OpStateUnknown,
				globalAst.ReferenceValidationSource: op.OpStateUnknown,
				globalAst.TerraformValidateSource:   op.OpStateUnknown,
			},
		},
		{
			path:            paths[1],
			MetaState:       op.OpStateUnknown,
			RefOriginsState: op.OpStateUnknown,
			RefTargetsState: op.OpStateUnknown,
			PolicyTestDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          op.OpStateUnknown,
				globalAst.SchemaValidationSource:    op.OpStateUnknown,
				globalAst.ReferenceValidationSource: op.OpStateUnknown,
				globalAst.TerraformValidateSource:   op.OpStateUnknown,
			},
		},
	}

	if diff := cmp.Diff(expectedRecords, list, cmpOpts); diff != "" {
		t.Fatalf("unexpected records in list: %s", diff)
	}
}

func TestPolicyStore_Remove(t *testing.T) {
	s, _ := setupStore(t)
	path := t.TempDir()
	_ = s.Add(path)

	if !s.Exists(path) {
		t.Fatal("expected record to exist")
	}

	_ = s.Remove(path)

	if s.Exists(path) {
		t.Fatal("expected record to be deleted")
	}
}
