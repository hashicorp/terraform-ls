package module

import (
	"context"
	"log"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
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

	log.Printf("Examining %v", modPath)
	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	err = ParseModuleConfiguration(fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, _ := ss.Modules.ModuleByPath(modPath)
	log.Printf("Stored: %v", mod.Path)
	log.Printf("Stored Meta: %#v", mod.Meta)

	err = GetModuleMetadataFromTFRegistry(ctx, ss.Modules, ss.RegistryModuleMetadataSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := tfaddr.ParseRawModuleSourceRegistry("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("0.0.8"))

	exists := ss.RegistryModuleMetadataSchemas.Exists(addr, cons)
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	declaredModule := module.DeclaredModuleCall{
		LocalName:  "ec",
		SourceAddr: addr,
		Version:    cons,
	}
	meta, err := ss.Modules.DeclaredModuleMeta(declaredModule)
	if err != nil {
		t.Fatal(err)
	}
	expectedMeta := &module.RegistryModuleMetadataSchema{
		Version: version.Must(version.NewVersion("0.0.8")),
		Inputs: []module.RegistryInput{
			{
				Name:        "autoscale",
				Type:        cty.String,
				Description: "Enable autoscaling of elasticsearch",
				Required:    false,
			},
			{
				Name:        "ec_stack_version",
				Type:        cty.String,
				Description: "Version of Elastic Cloud stack to deploy",
				Required:    false,
			},
			{
				Name:        "name",
				Type:        cty.String,
				Description: "Name of resources",
				Required:    false,
			},
			{
				Name:        "traffic_filter_sourceip",
				Type:        cty.String,
				Description: "traffic filter source IP",
				Required:    false,
			},
			{
				Name:        "ec_region",
				Type:        cty.String,
				Description: "cloud provider region",
				Required:    false,
			},
			{
				Name:        "deployment_templateid",
				Type:        cty.String,
				Description: "ID of Elastic Cloud deployment type",
				Required:    false,
			},
		},
		Outputs: []module.RegistryOutput{
			{
				Name:        "elasticsearch_password",
				Description: "elasticsearch password",
			},
			{
				Name:        "deployment_id",
				Description: "Elastic Cloud deployment ID",
			},
			{
				Name:        "elasticsearch_version",
				Description: "Stack version deployed",
			},
			{
				Name:        "elasticsearch_cloud_id",
				Description: "Elastic Cloud project deployment ID",
			},
			{
				Name:        "elasticsearch_https_endpoint",
				Description: "elasticsearch https endpoint",
			},
			{
				Name:        "elasticsearch_username",
				Description: "elasticsearch username",
			},
		},
	}

	log.Printf("Expected: %#v", expectedMeta.Inputs[0].Type)
	log.Printf("Actual: %#v", meta.Inputs[0].Type)
	if diff := cmp.Diff(expectedMeta, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}
