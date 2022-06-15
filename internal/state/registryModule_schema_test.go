package state

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
	"github.com/zclconf/go-cty/cty"
)

func TestStateStore_cache_metadata(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	source, err := tfaddr.ParseRawModuleSourceRegistry("terraform-aws-modules/eks/aws")
	if err != nil {
		t.Fatal(err)
	}

	v := version.Must(version.NewVersion("3.10.0"))
	c := version.MustConstraints(version.NewConstraint(">= 3.0"))
	inputs := []module.RegistryInput{
		{
			Name:        "foo",
			Type:        cty.String,
			Description: "baz",
			Default:     cty.StringVal("woot"),
			Required:    false,
		},
	}
	outputs := []module.RegistryOutput{
		{
			Name:        "wakka",
			Description: "fozzy",
		},
	}

	// should be false
	exists := s.RegistryModuleMetadataSchemas.Exists(source, c)
	if exists == true {
		t.Fatal("should not exist")
	}

	// store a dummy data
	err = s.RegistryModuleMetadataSchemas.Cache(source, v, inputs, outputs)
	if err != nil {
		t.Fatal(err)
	}

	// should be true
	exists = s.RegistryModuleMetadataSchemas.Exists(source, c)
	if exists != true {
		t.Fatal("should exist")
	}
}

func TestModule_DeclaredModuleMeta(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	source, err := tfaddr.ParseRawModuleSourceRegistry("terraform-aws-modules/eks/aws")
	if err != nil {
		t.Fatal(err)
	}

	v := version.Must(version.NewVersion("3.10.0"))
	inputs := []module.RegistryInput{
		{
			Name:        "foo",
			Type:        cty.String,
			Description: "baz",
			Default:     cty.StringVal("woot"),
			Required:    false,
		},
	}
	outputs := []module.RegistryOutput{
		{
			Name:        "wakka",
			Description: "fozzy",
		},
	}

	// store some dummy data
	err = ss.RegistryModuleMetadataSchemas.Cache(source, v, inputs, outputs)
	if err != nil {
		t.Fatal(err)
	}

	modCall := module.DeclaredModuleCall{
		LocalName:  "refname",
		SourceAddr: source,
		Version:    version.MustConstraints(version.NewConstraint(">= 3.0")),
	}
	meta, err := ss.Modules.DeclaredModuleMeta(modCall)
	if err != nil {
		t.Fatal(err)
	}

	expectedMeta := &module.RegistryModuleMetadataSchema{
		Version: v,
		Inputs:  []module.RegistryInput{},
		Outputs: []module.RegistryOutput{},
	}
	if diff := cmp.Diff(expectedMeta, meta); diff != "" {
		t.Fatalf("mismatch chached metadata: %s", diff)
	}
}
