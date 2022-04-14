package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

func Test_parseModuleRecords(t *testing.T) {
	tests := []struct {
		name    string
		records []datadir.ModuleRecord
		want    []moduleCall
	}{
		{
			name: "detects terraform module types",
			records: []datadir.ModuleRecord{
				{
					Key:        "ec2_instances",
					SourceAddr: "terraform-aws-modules/ec2-instance/aws",
					VersionStr: "2.12.0",
					Dir:        ".terraform\\modules\\ec2_instances",
				},
				{
					Key:        "web_server_sg",
					SourceAddr: "github.com/terraform-aws-modules/terraform-aws-security-group",
					VersionStr: "",
					Dir:        ".terraform\\modules\\web_server_sg",
				},
				{
					Key:        "eks",
					SourceAddr: "terraform-aws-modules/eks/aws",
					VersionStr: "17.20.0",
					Dir:        ".terraform\\modules\\eks",
				},
				{
					Key:        "eks.fargate",
					SourceAddr: "./modules/fargate",
					VersionStr: "",
					Dir:        ".terraform\\modules\\eks\\modules\\fargate",
				},
			},
			want: []moduleCall{
				{
					Name:             "ec2_instances",
					SourceAddr:       "terraform-aws-modules/ec2-instance/aws",
					Version:          "2.12.0",
					SourceType:       "tfregistry",
					DocsLink:         "https://registry.terraform.io/modules/terraform-aws-modules/ec2-instance/aws/2.12.0",
					DependentModules: []moduleCall{},
				},
				{
					Name:       "eks",
					SourceAddr: "terraform-aws-modules/eks/aws",
					Version:    "17.20.0",
					SourceType: "tfregistry",
					DocsLink:   "https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/17.20.0",
					DependentModules: []moduleCall{
						{
							Name:             "fargate",
							SourceAddr:       "./modules/fargate",
							Version:          "",
							SourceType:       "local",
							DocsLink:         "",
							DependentModules: []moduleCall{},
						},
					},
				},
				{
					Name:             "web_server_sg",
					SourceAddr:       "github.com/terraform-aws-modules/terraform-aws-security-group",
					Version:          "",
					SourceType:       "github",
					DocsLink:         "",
					DependentModules: []moduleCall{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseModuleRecords(tt.records)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("module mismatch: %s", diff)
			}
		})
	}
}

// With the release of Terraform 1.1.0 module source addresses are now stored normalized
func Test_parseModuleRecords_v1_1(t *testing.T) {
	tests := []struct {
		name    string
		records []datadir.ModuleRecord
		want    []moduleCall
	}{
		{
			name: "detects terraform module types",
			records: []datadir.ModuleRecord{
				{
					Key:        "ec2_instances",
					SourceAddr: "registry.terraform.io/terraform-aws-modules/ec2-instance/aws",
					VersionStr: "2.12.0",
					Dir:        ".terraform\\modules\\ec2_instances",
				},
			},
			want: []moduleCall{
				{
					Name:             "ec2_instances",
					SourceAddr:       "registry.terraform.io/terraform-aws-modules/ec2-instance/aws",
					Version:          "2.12.0",
					SourceType:       "tfregistry",
					DocsLink:         "https://registry.terraform.io/modules/terraform-aws-modules/ec2-instance/aws/2.12.0",
					DependentModules: []moduleCall{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseModuleRecords(tt.records)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("module mismatch: %s", diff)
			}
		})
	}
}
