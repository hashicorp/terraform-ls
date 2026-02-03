// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(datadir.ModuleManifest{}),
	cmp.AllowUnexported(hclsyntax.Body{}),
	cmp.Comparer(func(x, y version.Constraint) bool {
		return x.String() == y.String()
	}),
	cmp.Comparer(func(x, y hcl.File) bool {
		return (x.Body == y.Body &&
			cmp.Equal(x.Bytes, y.Bytes))
	}),
	ctydebug.CmpOptions,
}

// Test a scenario where Terraform 0.13+ produced schema with non-legacy
// addresses but lookup is still done via legacy address
func TestStateStore_IncompleteSchema_legacyLookup(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr := tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: "hashicorp",
		Type:      "aws",
	}
	pv := testVersion(t, "3.2.0")

	pvs := map[tfaddr.Provider]*version.Version{
		addr: pv,
	}

	// obtaining versions typically takes less time than schema itself
	// so we test that "incomplete" state is handled correctly too

	err = rs.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.13.0"), pvs, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gs.ProviderSchemas.ProviderSchema(modPath, globalState.NewLegacyProvider("aws"), testConstraint(t, ">= 1.0"))
	if err == nil {
		t.Fatal("expected error when requesting incomplete schema")
	}
	expectedErr := &globalState.NoSchemaError{}
	if !errors.As(err, &expectedErr) {
		t.Fatalf("unexpected error: %#v", err)
	}

	// next attempt (after schema is actually obtained) should not fail

	err = gs.ProviderSchemas.AddLocalSchema(modPath, addr, &tfschema.ProviderSchema{})
	if err != nil {
		t.Fatal(err)
	}

	ps, err := gs.ProviderSchemas.ProviderSchema(modPath, globalState.NewLegacyProvider("aws"), testConstraint(t, ">= 1.0"))
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("expected provider schema not to be nil")
	}
}

func TestStateStore_AddLocalSchema_duplicateWithVersion(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr := tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: "hashicorp",
		Type:      "aws",
	}
	schema := &tfschema.ProviderSchema{}

	err = gs.ProviderSchemas.AddLocalSchema(modPath, addr, schema)
	if err != nil {
		t.Fatal(err)
	}

	pv := map[tfaddr.Provider]*version.Version{
		addr: testVersion(t, "1.2.0"),
	}
	err = rs.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.12.0"), pv, nil)
	if err != nil {
		t.Fatal(err)
	}

	si, err := gs.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas := schemaSliceFromIterator(si)
	expectedSchemas := []*globalState.ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (0): %s", diff)
	}

	err = gs.ProviderSchemas.AddLocalSchema(modPath, addr, schema)
	if err != nil {
		t.Fatal(err)
	}

	si, err = gs.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas = schemaSliceFromIterator(si)
	expectedSchemas = []*globalState.ProviderSchema{
		{
			Address: addr,
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (1): %s", diff)
	}

	pv = map[tfaddr.Provider]*version.Version{
		addr: testVersion(t, "1.5.0"),
	}
	err = rs.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.12.0"), pv, nil)
	if err != nil {
		t.Fatal(err)
	}

	si, err = gs.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas = schemaSliceFromIterator(si)
	expectedSchemas = []*globalState.ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.5.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (2): %s", diff)
	}
}

func TestStateStore_AddLocalSchema_versionFirst(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr := tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: "hashicorp",
		Type:      "aws",
	}

	pv := map[tfaddr.Provider]*version.Version{
		addr: testVersion(t, "1.2.0"),
	}
	err = rs.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.12.0"), pv, nil)
	if err != nil {
		t.Fatal(err)
	}

	si, err := gs.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas := schemaSliceFromIterator(si)
	expectedSchemas := []*globalState.ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (1): %s", diff)
	}

	err = gs.ProviderSchemas.AddLocalSchema(modPath, addr, &tfschema.ProviderSchema{})
	if err != nil {
		t.Fatal(err)
	}

	si, err = gs.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas = schemaSliceFromIterator(si)
	expectedSchemas = []*globalState.ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: &tfschema.ProviderSchema{},
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (2): %s", diff)
	}
}

func TestStateStore_ProviderSchema_localHasPriority(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*globalState.ProviderSchema{
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			Version: testVersion(t, "0.9.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/blah 0.9.0"),
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			Version: testVersion(t, "0.9.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 0.9.0"),
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			Version: testVersion(t, "1.0.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("local: hashicorp/aws 1.0.0"),
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			Version: testVersion(t, "1.0.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 1.0.0"),
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			Version: testVersion(t, "1.3.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 1.3.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, gs.ProviderSchemas, rs, ps)
	}

	ps, err := gs.ProviderSchemas.ProviderSchema(modPath,
		tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "aws"),
		testConstraint(t, "1.0.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("no schema found")
	}

	expectedDescription := "local: hashicorp/aws 1.0.0"
	if ps.Provider.Description.Value != expectedDescription {
		t.Fatalf("description doesn't match. expected: %q, got: %q",
			expectedDescription, ps.Provider.Description.Value)
	}
}

func TestStateStore_ProviderSchema_legacyAddress_exactMatch(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*globalState.ProviderSchema{
		{
			Address: globalState.NewLegacyProvider("aws"),
			Version: testVersion(t, "2.0.0"),
			Source:  globalState.LocalSchemaSource{ModulePath: modPath},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("local: -/aws 2.0.0"),
				},
			},
		},
		{
			Address: globalState.NewDefaultProvider("aws"),
			Version: testVersion(t, "2.5.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 2.5.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, gs.ProviderSchemas, rs, ps)
	}

	ps, err := gs.ProviderSchemas.ProviderSchema(modPath,
		globalState.NewLegacyProvider("aws"),
		testConstraint(t, "2.0.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("no schema found")
	}

	expectedDescription := "local: -/aws 2.0.0"
	if ps.Provider.Description.Value != expectedDescription {
		t.Fatalf("description doesn't match. expected: %q, got: %q",
			expectedDescription, ps.Provider.Description.Value)
	}

	// Check that detail has legacy namespace in detail, but no link
	expectedDetail := "-/aws 2.0.0"
	if ps.Provider.Detail != expectedDetail {
		t.Fatalf("detail doesn't match. expected: %q, got: %q",
			expectedDetail, ps.Provider.Detail)
	}
	if ps.Provider.DocsLink != nil {
		t.Fatalf("docs link is not empty, got: %#v",
			ps.Provider.DocsLink)
	}
}

func TestStateStore_ProviderSchema_legacyAddress_looseMatch(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*globalState.ProviderSchema{
		{
			Address: globalState.NewDefaultProvider("aws"),
			Version: testVersion(t, "2.5.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 2.5.0"),
				},
			},
		},
		{
			Address: tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "grafana", "grafana"),
			Version: testVersion(t, "1.0.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: grafana/grafana 1.0.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, gs.ProviderSchemas, rs, ps)
	}

	ps, err := gs.ProviderSchemas.ProviderSchema(modPath,
		globalState.NewLegacyProvider("grafana"),
		testConstraint(t, "1.0.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("no schema found")
	}

	expectedDescription := "preload: grafana/grafana 1.0.0"
	if ps.Provider.Description.Value != expectedDescription {
		t.Fatalf("description doesn't match. expected: %q, got: %q",
			expectedDescription, ps.Provider.Description.Value)
	}
}

func TestStateStore_ProviderSchema_terraformProvider(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*globalState.ProviderSchema{
		{
			Address: globalState.NewBuiltInProvider("terraform"),
			Version: testVersion(t, "1.0.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: builtin/terraform 1.0.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, gs.ProviderSchemas, rs, ps)
	}

	ps, err := gs.ProviderSchemas.ProviderSchema(modPath,
		globalState.NewLegacyProvider("terraform"),
		testConstraint(t, "1.0.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("no schema found")
	}

	expectedDescription := "preload: builtin/terraform 1.0.0"
	if ps.Provider.Description.Value != expectedDescription {
		t.Fatalf("description doesn't match. expected: %q, got: %q",
			expectedDescription, ps.Provider.Description.Value)
	}

	ps, err = gs.ProviderSchemas.ProviderSchema(modPath,
		globalState.NewDefaultProvider("terraform"),
		testConstraint(t, "1.0.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("no schema found")
	}

	expectedDescription = "preload: builtin/terraform 1.0.0"
	if ps.Provider.Description.Value != expectedDescription {
		t.Fatalf("description doesn't match. expected: %q, got: %q",
			expectedDescription, ps.Provider.Description.Value)
	}
}

func TestStateStore_ListSchemas(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPathA := filepath.Join("special", "moduleA")
	err = rs.Add(modPathA)
	if err != nil {
		t.Fatal(err)
	}
	modPathB := filepath.Join("special", "moduleB")
	err = rs.Add(modPathB)
	if err != nil {
		t.Fatal(err)
	}
	modPathC := filepath.Join("special", "moduleC")
	err = rs.Add(modPathC)
	if err != nil {
		t.Fatal(err)
	}

	localSchemas := []*globalState.ProviderSchema{
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			Version: testVersion(t, "1.0.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathA,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			Version: testVersion(t, "0.9.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathA,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			Version: testVersion(t, "1.0.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathB,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			Version: testVersion(t, "1.3.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathC,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
	}
	for _, ps := range localSchemas {
		addAnySchema(t, gs.ProviderSchemas, rs, ps)
	}

	si, err := gs.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}

	schemas := schemaSliceFromIterator(si)

	expectedSchemas := []*globalState.ProviderSchema{
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			Version: testVersion(t, "0.9.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathA,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Detail:   "hashicorp/aws-local 0.9.0",
					HoverURL: "https://registry.terraform.io/providers/hashicorp/aws-local/0.9.0/docs",
					DocsLink: &schema.DocsLink{
						URL:     "https://registry.terraform.io/providers/hashicorp/aws-local/0.9.0/docs",
						Tooltip: "hashicorp/aws-local Documentation",
					},
					Attributes: map[string]*schema.AttributeSchema{},
					Blocks:     map[string]*schema.BlockSchema{},
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			Version: testVersion(t, "1.0.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathB,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Detail:   "hashicorp/aws-local 1.0.0",
					HoverURL: "https://registry.terraform.io/providers/hashicorp/aws-local/1.0.0/docs",
					DocsLink: &schema.DocsLink{
						URL:     "https://registry.terraform.io/providers/hashicorp/aws-local/1.0.0/docs",
						Tooltip: "hashicorp/aws-local Documentation",
					},
					Attributes: map[string]*schema.AttributeSchema{},
					Blocks:     map[string]*schema.BlockSchema{},
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			Version: testVersion(t, "1.3.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathC,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Detail:   "hashicorp/aws-local 1.3.0",
					HoverURL: "https://registry.terraform.io/providers/hashicorp/aws-local/1.3.0/docs",
					DocsLink: &schema.DocsLink{
						URL:     "https://registry.terraform.io/providers/hashicorp/aws-local/1.3.0/docs",
						Tooltip: "hashicorp/aws-local Documentation",
					},
					Attributes: map[string]*schema.AttributeSchema{},
					Blocks:     map[string]*schema.BlockSchema{},
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			Version: testVersion(t, "1.0.0"),
			Source: globalState.LocalSchemaSource{
				ModulePath: modPathA,
			},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Detail:   "hashicorp/blah 1.0.0",
					HoverURL: "https://registry.terraform.io/providers/hashicorp/blah/1.0.0/docs",
					DocsLink: &schema.DocsLink{
						URL:     "https://registry.terraform.io/providers/hashicorp/blah/1.0.0/docs",
						Tooltip: "hashicorp/blah Documentation",
					},
					Attributes: map[string]*schema.AttributeSchema{},
					Blocks:     map[string]*schema.BlockSchema{},
				},
			},
		},
	}
	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas: %s", diff)
	}
}

// BenchmarkProviderSchema exercises the hot path for most handlers which require schema
func BenchmarkProviderSchema(b *testing.B) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		b.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		b.Fatal(err)
	}

	schemas := []*globalState.ProviderSchema{
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			Version: testVersion(b, "0.9.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/blah 0.9.0"),
				},
			},
		},
		{
			Address: tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			Version: testVersion(b, "0.9.0"),
			Source:  globalState.PreloadedSchemaSource{},
			Schema: &tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 0.9.0"),
				},
			},
		},
	}
	for _, ps := range schemas {
		addAnySchema(b, gs.ProviderSchemas, rs, ps)
	}

	expectedPs := &tfschema.ProviderSchema{
		Provider: &schema.BodySchema{
			Description: lang.PlainText("preload: hashicorp/aws 0.9.0"),
		},
	}
	for n := 0; n < b.N; n++ {
		foundPs, err := gs.ProviderSchemas.ProviderSchema("/test", globalState.NewDefaultProvider("aws"), testConstraint(b, "0.9.0"))
		if err != nil {
			b.Fatal(err)
		}
		if diff := cmp.Diff(expectedPs, foundPs, cmpOpts); diff != "" {
			b.Fatalf("schema doesn't match: %s", diff)
		}
	}
}

func schemaSliceFromIterator(it *globalState.ProviderSchemaIterator) []*globalState.ProviderSchema {
	schemas := make([]*globalState.ProviderSchema, 0)
	for ps := it.Next(); ps != nil; ps = it.Next() {
		schemas = append(schemas, ps.Copy())
	}
	return schemas
}

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func addAnySchema(t testOrBench, ss *globalState.ProviderSchemaStore, rs *state.RootStore, ps *globalState.ProviderSchema) {
	switch s := ps.Source.(type) {
	case globalState.PreloadedSchemaSource:
		err := ss.AddPreloadedSchema(ps.Address, ps.Version, ps.Schema)
		if err != nil {
			t.Fatal(err)
		}
	case globalState.LocalSchemaSource:
		err := ss.AddLocalSchema(s.ModulePath, ps.Address, ps.Schema)
		if err != nil {
			t.Fatal(err)

		}
		pVersions := map[tfaddr.Provider]*version.Version{
			ps.Address: ps.Version,
		}
		err = rs.UpdateTerraformAndProviderVersions(s.ModulePath, testVersion(t, "0.14.0"), pVersions, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testVersion(t testOrBench, v string) *version.Version {
	ver, err := version.NewVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return ver
}

func testConstraint(t testOrBench, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}
