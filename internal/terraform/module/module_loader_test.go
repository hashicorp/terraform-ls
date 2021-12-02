package module

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestModuleLoader_referenceCollection(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	fs := filesystem.NewFilesystem()

	ml := newModuleLoader(fs, ss.Modules, ss.ProviderSchemas)
	ml.logger = testLogger()

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join(testData, "single-root-no-modules")

	ss.Modules.Add(modPath)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancelFunc)

	ml.Start(ctx)

	modOp := NewModuleOperation(modPath, op.OpTypeParseModuleConfiguration)
	err = ml.EnqueueModuleOp(modOp)
	if err != nil {
		t.Fatal(err)
	}
	<-modOp.done()

	manifestOp := NewModuleOperation(modPath, op.OpTypeLoadModuleMetadata)
	err = ml.EnqueueModuleOp(manifestOp)
	if err != nil {
		t.Fatal(err)
	}

	originsOp := NewModuleOperation(modPath, op.OpTypeDecodeReferenceOrigins)
	err = ml.EnqueueModuleOp(originsOp)
	if err != nil {
		t.Fatal(err)
	}
	targetsOp := NewModuleOperation(modPath, op.OpTypeDecodeReferenceTargets)
	err = ml.EnqueueModuleOp(targetsOp)
	if err != nil {
		t.Fatal(err)
	}
	varsOriginsOp := NewModuleOperation(modPath, op.OpTypeDecodeVarsReferences)
	err = ml.EnqueueModuleOp(varsOriginsOp)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("waiting for all operations to finish")
	<-manifestOp.done()
	t.Log("manifest parsed")
	<-originsOp.done()
	t.Log("origins collected")
	<-targetsOp.done()
	t.Log("targets collected")
	<-varsOriginsOp.done()
	t.Log("vars origins collected")

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedOrigins := reference.Origins{
		reference.LocalOrigin{
			Addr: lang.Address{lang.RootStep{Name: "var"}, lang.AttrStep{Name: "count"}},
			Range: hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 14, Column: 11, Byte: 184},
				End:      hcl.Pos{Line: 14, Column: 20, Byte: 193},
			},
			Constraints: []reference.OriginConstraint{
				{
					OfType: cty.DynamicPseudoType,
				},
			},
		},
	}
	if diff := cmp.Diff(expectedOrigins, mod.RefOrigins, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("unexpected origins: %s", diff)
	}

	expectedTargets := reference.Targets{
		{
			Addr: lang.Address{
				lang.RootStep{Name: "output"},
				lang.AttrStep{Name: "pet_count"},
			},
			ScopeId: lang.ScopeId("output"),
			RangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 13, Column: 1, Byte: 153},
				End:      hcl.Pos{Line: 15, Column: 2, Byte: 195},
			},
			DefRangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 13, Column: 1, Byte: 153},
				End:      hcl.Pos{Line: 13, Column: 19, Byte: 171},
			},
			Name: "output",
		},
		{
			Addr: lang.Address{
				lang.RootStep{Name: "random_pet"},
				lang.AttrStep{Name: "application"},
			},
			ScopeId: lang.ScopeId("resource"),
			RangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 6, Column: 2, Byte: 99},
			},
			DefRangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 36, Byte: 35},
			},
			Name: "resource",
		},
		{
			Addr: lang.Address{
				lang.RootStep{Name: "var"},
				lang.AttrStep{Name: "count"},
			},
			ScopeId: lang.ScopeId("variable"),
			RangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 8, Column: 1, Byte: 101},
				End:      hcl.Pos{Line: 11, Column: 2, Byte: 151},
			},
			DefRangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 8, Column: 1, Byte: 101},
				End:      hcl.Pos{Line: 8, Column: 17, Byte: 117},
			},
			Name: "variable",
		},
		{
			Addr: lang.Address{
				lang.RootStep{Name: "var"},
				lang.AttrStep{Name: "count"},
			},
			ScopeId: lang.ScopeId("variable"),
			RangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 8, Column: 1, Byte: 101},
				End:      hcl.Pos{Line: 11, Column: 2, Byte: 151},
			},
			DefRangePtr: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 8, Column: 1, Byte: 101},
				End:      hcl.Pos{Line: 8, Column: 17, Byte: 117},
			},
			Type: cty.Number,
		},
	}
	expectedTargets = append(expectedTargets, builtinReferences(modPath)...)
	if diff := cmp.Diff(expectedTargets, mod.RefTargets, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("unexpected targets: %s", diff)
	}
}
