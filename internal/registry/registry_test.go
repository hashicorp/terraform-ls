package registry

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
)

func TestGetTFRegistryInfo(t *testing.T) {
	addr, err := tfaddr.ParseRawModuleSourceRegistry("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}

	modCall := module.DeclaredModuleCall{
		LocalName:  "refname",
		SourceAddr: addr,
		Version:    version.MustConstraints(version.NewConstraint("0.0.8")),
	}

	data, err := GetTFRegistryInfo(addr, modCall)
	if err != nil {
		t.Fatal(err)
	}
	expectedData := &TerraformRegistryModule{
		Version: "0.0.8",
		Root: ModuleRoot{
			Inputs: []Input{
				{
					Name:        "autoscale",
					Type:        "string",
					Description: "Enable autoscaling of elasticsearch",
					Default:     "\"true\"",
					Required:    false,
				},
				{
					Name:        "ec_stack_version",
					Type:        "string",
					Description: "Version of Elastic Cloud stack to deploy",
					Default:     "\"\"",
					Required:    false,
				},
				{
					Name:        "name",
					Type:        "string",
					Description: "Name of resources",
					Default:     "\"ecproject\"",
					Required:    false,
				},
				{
					Name:        "traffic_filter_sourceip",
					Type:        "string",
					Description: "traffic filter source IP",
					Default:     "\"\"",
					Required:    false,
				},
				{
					Name:        "ec_region",
					Type:        "string",
					Description: "cloud provider region",
					Default:     "\"gcp-us-west1\"",
					Required:    false,
				},
				{
					Name:        "deployment_templateid",
					Type:        "string",
					Description: "ID of Elastic Cloud deployment type",
					Default:     "\"gcp-io-optimized\"",
					Required:    false,
				},
			},
			Outputs: []Output{
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
		},
	}
	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("mismatched data: %s", diff)
	}
}

func TestGetVersion(t *testing.T) {
	addr, err := tfaddr.ParseRawModuleSourceRegistry("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("0.0.8"))
	v, err := GetVersion(addr, cons)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersion := version.Must(version.NewVersion("0.0.8"))
	if !expectedVersion.Equal(v) {
		t.Fatalf("expected version: %s, given: %s", expectedVersion, v)
	}
}
