package module

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func TestGetModuleMetadataFromTFRegistry(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-external-module")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = GetModuleMetadataFromTFRegistry(ctx, ss.Modules, ss.RegistryModuleMetadataSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := tfaddr.ParseRawModuleSourceRegistry("terraform-aws-modules/eks/aws")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("18.23.0"))

	exists := ss.RegistryModuleMetadataSchemas.Exists(addr, cons)
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}
}
