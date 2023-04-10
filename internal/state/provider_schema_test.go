// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func TestStateStore_AddPreloadedSchema_duplicate(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	addr := tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: "hashicorp",
		Type:      "aws",
	}
	pv := testVersion(t, "1.0.0")
	schema := &tfschema.ProviderSchema{}

	err = s.ProviderSchemas.AddPreloadedSchema(addr, pv, schema)
	if err != nil {
		t.Fatal(err)
	}

	err = s.ProviderSchemas.AddPreloadedSchema(addr, pv, schema)
	if err == nil {
		t.Fatal("expected duplicate insertion to fail")
	}

	aeErr := &AlreadyExistsError{}
	if !errors.As(err, &aeErr) {
		t.Fatalf("unexpected error: %s", err)
	}
}

// Test a scenario where Terraform 0.13+ produced schema with non-legacy
// addresses but lookup is still done via legacy address
func TestStateStore_IncompleteSchema_legacyLookup(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	err = s.Modules.Add(modPath)
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

	err = s.Modules.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.13.0"), pvs, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.ProviderSchemas.ProviderSchema(modPath, NewLegacyProvider("aws"), testConstraint(t, ">= 1.0"))
	if err == nil {
		t.Fatal("expected error when requesting incomplete schema")
	}
	expectedErr := &NoSchemaError{}
	if !errors.As(err, &expectedErr) {
		t.Fatalf("unexpected error: %#v", err)
	}

	// next attempt (after schema is actually obtained) should not fail

	err = s.ProviderSchemas.AddLocalSchema(modPath, addr, &tfschema.ProviderSchema{})
	if err != nil {
		t.Fatal(err)
	}

	ps, err := s.ProviderSchemas.ProviderSchema(modPath, NewLegacyProvider("aws"), testConstraint(t, ">= 1.0"))
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("expected provider schema not to be nil")
	}
}

func TestStateStore_AddLocalSchema_duplicate(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	addr := tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: "hashicorp",
		Type:      "aws",
	}
	schema := &tfschema.ProviderSchema{}

	err = s.ProviderSchemas.AddLocalSchema(modPath, addr, schema)
	if err != nil {
		t.Fatal(err)
	}

	err = s.ProviderSchemas.AddLocalSchema(modPath, addr, schema)
	if err != nil {
		t.Fatal(err)
	}

	si, err := s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas := schemaSliceFromIterator(si)
	expectedSchemas := []*ProviderSchema{
		{
			Address: addr,
			Source: LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas: %s", diff)
	}
}

func TestStateStore_AddLocalSchema_duplicateWithVersion(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr := tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: "hashicorp",
		Type:      "aws",
	}
	schema := &tfschema.ProviderSchema{}

	err = s.ProviderSchemas.AddLocalSchema(modPath, addr, schema)
	if err != nil {
		t.Fatal(err)
	}

	pv := map[tfaddr.Provider]*version.Version{
		addr: testVersion(t, "1.2.0"),
	}
	err = s.Modules.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.12.0"), pv, nil)
	if err != nil {
		t.Fatal(err)
	}

	si, err := s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas := schemaSliceFromIterator(si)
	expectedSchemas := []*ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (0): %s", diff)
	}

	err = s.ProviderSchemas.AddLocalSchema(modPath, addr, schema)
	if err != nil {
		t.Fatal(err)
	}

	si, err = s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas = schemaSliceFromIterator(si)
	expectedSchemas = []*ProviderSchema{
		{
			Address: addr,
			Source: LocalSchemaSource{
				ModulePath: modPath,
			},
			Schema: schema,
		},
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: LocalSchemaSource{
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
	err = s.Modules.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.12.0"), pv, nil)
	if err != nil {
		t.Fatal(err)
	}

	si, err = s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas = schemaSliceFromIterator(si)
	expectedSchemas = []*ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.5.0"),
			Source: LocalSchemaSource{
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
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Modules.Add(modPath)
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
	err = s.Modules.UpdateTerraformAndProviderVersions(modPath, testVersion(t, "0.12.0"), pv, nil)
	if err != nil {
		t.Fatal(err)
	}

	si, err := s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas := schemaSliceFromIterator(si)
	expectedSchemas := []*ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: LocalSchemaSource{
				ModulePath: modPath,
			},
		},
	}

	if diff := cmp.Diff(expectedSchemas, schemas, cmpOpts); diff != "" {
		t.Fatalf("unexpected schemas (1): %s", diff)
	}

	err = s.ProviderSchemas.AddLocalSchema(modPath, addr, &tfschema.ProviderSchema{})
	if err != nil {
		t.Fatal(err)
	}

	si, err = s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schemas = schemaSliceFromIterator(si)
	expectedSchemas = []*ProviderSchema{
		{
			Address: addr,
			Version: testVersion(t, "1.2.0"),
			Source: LocalSchemaSource{
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
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*ProviderSchema{
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			testVersion(t, "0.9.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/blah 0.9.0"),
				},
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			testVersion(t, "0.9.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 0.9.0"),
				},
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			testVersion(t, "1.0.0"),
			LocalSchemaSource{
				ModulePath: modPath,
			},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("local: hashicorp/aws 1.0.0"),
				},
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			testVersion(t, "1.0.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 1.0.0"),
				},
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			testVersion(t, "1.3.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 1.3.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, s.ProviderSchemas, s.Modules, ps)
	}

	ps, err := s.ProviderSchemas.ProviderSchema(modPath,
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
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*ProviderSchema{
		{
			NewLegacyProvider("aws"),
			testVersion(t, "2.0.0"),
			LocalSchemaSource{ModulePath: modPath},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("local: -/aws 2.0.0"),
				},
			},
		},
		{
			NewDefaultProvider("aws"),
			testVersion(t, "2.5.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 2.5.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, s.ProviderSchemas, s.Modules, ps)
	}

	ps, err := s.ProviderSchemas.ProviderSchema(modPath,
		NewLegacyProvider("aws"),
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
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*ProviderSchema{
		{
			NewDefaultProvider("aws"),
			testVersion(t, "2.5.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 2.5.0"),
				},
			},
		},
		{
			tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "grafana", "grafana"),
			testVersion(t, "1.0.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: grafana/grafana 1.0.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, s.ProviderSchemas, s.Modules, ps)
	}

	ps, err := s.ProviderSchemas.ProviderSchema(modPath,
		NewLegacyProvider("grafana"),
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
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join("special", "module")
	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	schemas := []*ProviderSchema{
		{
			NewBuiltInProvider("terraform"),
			testVersion(t, "1.0.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: builtin/terraform 1.0.0"),
				},
			},
		},
	}

	for _, ps := range schemas {
		addAnySchema(t, s.ProviderSchemas, s.Modules, ps)
	}

	ps, err := s.ProviderSchemas.ProviderSchema(modPath,
		NewLegacyProvider("terraform"),
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

	ps, err = s.ProviderSchemas.ProviderSchema(modPath,
		NewDefaultProvider("terraform"),
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
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPathA := filepath.Join("special", "moduleA")
	err = s.Modules.Add(modPathA)
	if err != nil {
		t.Fatal(err)
	}
	modPathB := filepath.Join("special", "moduleB")
	err = s.Modules.Add(modPathB)
	if err != nil {
		t.Fatal(err)
	}
	modPathC := filepath.Join("special", "moduleC")
	err = s.Modules.Add(modPathC)
	if err != nil {
		t.Fatal(err)
	}

	localSchemas := []*ProviderSchema{
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			testVersion(t, "1.0.0"),
			LocalSchemaSource{
				ModulePath: modPathA,
			},
			&tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			testVersion(t, "0.9.0"),
			LocalSchemaSource{
				ModulePath: modPathA,
			},
			&tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			testVersion(t, "1.0.0"),
			LocalSchemaSource{
				ModulePath: modPathB,
			},
			&tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			testVersion(t, "1.3.0"),
			LocalSchemaSource{
				ModulePath: modPathC,
			},
			&tfschema.ProviderSchema{
				Provider: schema.NewBodySchema(),
			},
		},
	}
	for _, ps := range localSchemas {
		addAnySchema(t, s.ProviderSchemas, s.Modules, ps)
	}

	si, err := s.ProviderSchemas.ListSchemas()
	if err != nil {
		t.Fatal(err)
	}

	schemas := schemaSliceFromIterator(si)

	expectedSchemas := []*ProviderSchema{
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			testVersion(t, "0.9.0"),
			LocalSchemaSource{
				ModulePath: modPathA,
			},
			&tfschema.ProviderSchema{
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
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			testVersion(t, "1.0.0"),
			LocalSchemaSource{
				ModulePath: modPathB,
			},
			&tfschema.ProviderSchema{
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
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws-local",
			},
			testVersion(t, "1.3.0"),
			LocalSchemaSource{
				ModulePath: modPathC,
			},
			&tfschema.ProviderSchema{
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
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			testVersion(t, "1.0.0"),
			LocalSchemaSource{
				ModulePath: modPathA,
			},
			&tfschema.ProviderSchema{
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

func TestAllSchemasExist(t *testing.T) {
	testCases := []struct {
		Name               string
		Requirements       map[tfaddr.Provider]version.Constraints
		InstalledProviders InstalledProviders
		ExpectedMatch      bool
		ExpectedErr        bool
	}{
		{
			Name:               "empty requirements",
			Requirements:       map[tfaddr.Provider]version.Constraints{},
			InstalledProviders: InstalledProviders{},
			ExpectedMatch:      true,
			ExpectedErr:        false,
		},
		{
			Name: "missing all installed providers",
			Requirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.MustParseProviderSource("hashicorp/test"): version.MustConstraints(version.NewConstraint("1.0.0")),
			},
			InstalledProviders: InstalledProviders{},
			ExpectedMatch:      false,
			ExpectedErr:        false,
		},
		{
			Name: "missing one of two installed providers",
			Requirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.MustParseProviderSource("hashicorp/aws"):    version.MustConstraints(version.NewConstraint(">= 1.0.0")),
				tfaddr.MustParseProviderSource("hashicorp/google"): version.MustConstraints(version.NewConstraint(">= 1.0.0")),
			},
			InstalledProviders: InstalledProviders{
				tfaddr.MustParseProviderSource("hashicorp/aws"): version.Must(version.NewVersion("1.0.0")),
			},
			ExpectedMatch: false,
			ExpectedErr:   false,
		},
		{
			Name: "missing installed provider version",
			Requirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.MustParseProviderSource("hashicorp/aws"): version.MustConstraints(version.NewConstraint(">= 1.0.0")),
			},
			InstalledProviders: InstalledProviders{
				tfaddr.MustParseProviderSource("hashicorp/aws"): version.Must(version.NewVersion("0.1.0")),
			},
			ExpectedMatch: false,
			ExpectedErr:   false,
		},
		{
			Name: "matching installed providers",
			Requirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.MustParseProviderSource("hashicorp/test"): version.MustConstraints(version.NewConstraint("1.0.0")),
			},
			InstalledProviders: InstalledProviders{
				tfaddr.MustParseProviderSource("hashicorp/test"): version.Must(version.NewVersion("1.0.0")),
			},
			ExpectedMatch: true,
			ExpectedErr:   false,
		},
		{
			Name: "missing provider version in schema store",
			Requirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.MustParseProviderSource("hashicorp/test"): version.MustConstraints(version.NewConstraint(">=1.0.0")),
			},
			InstalledProviders: InstalledProviders{
				tfaddr.MustParseProviderSource("hashicorp/test"): nil,
			},
			ExpectedMatch: false,
			ExpectedErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ss, err := NewStateStore()
			if err != nil {
				t.Fatal(err)
			}

			for pAddr, pVersion := range tc.InstalledProviders {
				err = ss.ProviderSchemas.AddPreloadedSchema(pAddr, pVersion, &tfschema.ProviderSchema{})
				if err != nil {
					t.Fatal(err)
				}
			}

			exist, err := ss.ProviderSchemas.AllSchemasExist(tc.Requirements)
			if err != nil && !tc.ExpectedErr {
				t.Fatal(err)
			}
			if err == nil && tc.ExpectedErr {
				t.Fatal("expected error")
			}
			if exist && !tc.ExpectedMatch {
				t.Fatalf("expected schemas mismatch\nrequirements: %v\ninstalled: %v\n",
					tc.Requirements, tc.InstalledProviders)
			}
			if !exist && tc.ExpectedMatch {
				t.Fatalf("expected schemas match\nrequirements: %v\ninstalled: %v\n",
					tc.Requirements, tc.InstalledProviders)
			}
		})
	}
}

// BenchmarkProviderSchema exercises the hot path for most handlers which require schema
func BenchmarkProviderSchema(b *testing.B) {
	s, err := NewStateStore()
	if err != nil {
		b.Fatal(err)
	}

	schemas := []*ProviderSchema{
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "blah",
			},
			testVersion(b, "0.9.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/blah 0.9.0"),
				},
			},
		},
		{
			tfaddr.Provider{
				Hostname:  tfaddr.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "aws",
			},
			testVersion(b, "0.9.0"),
			PreloadedSchemaSource{},
			&tfschema.ProviderSchema{
				Provider: &schema.BodySchema{
					Description: lang.PlainText("preload: hashicorp/aws 0.9.0"),
				},
			},
		},
	}
	for _, ps := range schemas {
		addAnySchema(b, s.ProviderSchemas, s.Modules, ps)
	}

	expectedPs := &tfschema.ProviderSchema{
		Provider: &schema.BodySchema{
			Description: lang.PlainText("preload: hashicorp/aws 0.9.0"),
		},
	}
	for n := 0; n < b.N; n++ {
		foundPs, err := s.ProviderSchemas.ProviderSchema("/test", NewDefaultProvider("aws"), testConstraint(b, "0.9.0"))
		if err != nil {
			b.Fatal(err)
		}
		if diff := cmp.Diff(expectedPs, foundPs, cmpOpts); diff != "" {
			b.Fatalf("schema doesn't match: %s", diff)
		}
	}
}

func schemaSliceFromIterator(it *ProviderSchemaIterator) []*ProviderSchema {
	schemas := make([]*ProviderSchema, 0)
	for ps := it.Next(); ps != nil; ps = it.Next() {
		schemas = append(schemas, ps.Copy())
	}
	return schemas
}

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func addAnySchema(t testOrBench, ss *ProviderSchemaStore, ms *ModuleStore, ps *ProviderSchema) {
	switch s := ps.Source.(type) {
	case PreloadedSchemaSource:
		err := ss.AddPreloadedSchema(ps.Address, ps.Version, ps.Schema)
		if err != nil {
			t.Fatal(err)
		}
	case LocalSchemaSource:
		err := ss.AddLocalSchema(s.ModulePath, ps.Address, ps.Schema)
		if err != nil {
			t.Fatal(err)

		}
		pVersions := map[tfaddr.Provider]*version.Version{
			ps.Address: ps.Version,
		}
		err = ms.UpdateTerraformAndProviderVersions(s.ModulePath, testVersion(t, "0.14.0"), pVersions, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}
