package state

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/registry"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func TestStateStore_cache_metadata(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	source, _ := tfaddr.ParseRawModuleSourceRegistry("terraform-aws-modules/eks/aws")

	// source := tfaddr.ModuleSourceRegistry{
	// 	PackageAddr: tfaddr.ModuleRegistryPackage{
	// 		Host:         svchost.Hostname("registry.terraform.io"),
	// 		Namespace:    "hashicorp",
	// 		Name:         "subnets",
	// 		TargetSystem: "cidr",
	// 	},
	// 	Subdir: "foo",
	// }
	v := version.Must(version.NewVersion("3.10.0"))
	c, _ := version.NewConstraint(">= 3.0")
	inputs := []registry.Input{
		{
			Name:        "foo",
			Type:        "bar",
			Description: "baz",
			Default:     "woot",
			Required:    false,
		},
	}
	outputs := []registry.Output{
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
