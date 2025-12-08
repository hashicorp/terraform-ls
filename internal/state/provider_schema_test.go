// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
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

func testVersion(t testOrBench, v string) *version.Version {
	ver, err := version.NewVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return ver
}
