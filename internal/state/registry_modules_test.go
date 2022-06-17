package state

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/registry"
	"github.com/zclconf/go-cty-debug/ctydebug"
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
	inputs := []registry.Input{
		{
			Name:        "foo",
			Type:        cty.String,
			Description: lang.Markdown("baz"),
			Default:     cty.StringVal("woot"),
			Required:    false,
		},
	}
	outputs := []registry.Output{
		{
			Name:        "wakka",
			Description: lang.Markdown("fozzy"),
		},
	}

	// should be false
	exists, err := s.RegistryModules.Exists(source, c)
	if err != nil {
		t.Fatal(err)
	}
	if exists == true {
		t.Fatal("should not exist")
	}

	// store a dummy data
	err = s.RegistryModules.Cache(source, v, inputs, outputs)
	if err != nil {
		t.Fatal(err)
	}

	// should be true
	exists, err = s.RegistryModules.Exists(source, c)
	if err != nil {
		t.Fatal(err)
	}
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
	inputs := []registry.Input{
		{
			Name:        "foo",
			Type:        cty.String,
			Description: lang.Markdown("baz"),
			Default:     cty.StringVal("woot"),
			Required:    false,
		},
	}
	outputs := []registry.Output{
		{
			Name:        "wakka",
			Description: lang.Markdown("fozzy"),
		},
	}

	// store some dummy data
	err = ss.RegistryModules.Cache(source, v, inputs, outputs)
	if err != nil {
		t.Fatal(err)
	}

	cons := version.MustConstraints(version.NewConstraint(">= 3.0"))
	meta, err := ss.Modules.RegistryModuleMeta(source, cons)
	if err != nil {
		t.Fatal(err)
	}

	expectedMeta := &registry.ModuleData{
		Version: v,
		Inputs:  inputs,
		Outputs: outputs,
	}
	if diff := cmp.Diff(expectedMeta, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("mismatch chached metadata: %s", diff)
	}
}
