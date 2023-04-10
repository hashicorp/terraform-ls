// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

func Test_parseModuleRecords(t *testing.T) {
	tests := []struct {
		name        string
		moduleCalls tfmod.ModuleCalls
		want        []moduleCall
	}{
		{
			name: "detects terraform module types",
			moduleCalls: tfmod.ModuleCalls{
				Installed: map[string]tfmod.InstalledModuleCall{},
				Declared: map[string]tfmod.DeclaredModuleCall{
					"ec2_instances": {
						LocalName:  "ec2_instances",
						SourceAddr: tfaddr.MustParseModuleSource("terraform-aws-modules/ec2-instance/aws"),
						Version:    version.MustConstraints(version.NewConstraint("2.12.0")),
					},
					"web_server_sg": {
						LocalName:  "web_server_sg",
						SourceAddr: module.UnknownSourceAddr("github.com/terraform-aws-modules/terraform-aws-security-group"),
						Version:    nil,
					},
					"eks": {
						LocalName:  "eks",
						SourceAddr: tfaddr.MustParseModuleSource("terraform-aws-modules/eks/aws"),
						Version:    version.MustConstraints(version.NewConstraint("17.20.0")),
					},
					"beta": {
						LocalName:  "beta",
						SourceAddr: module.LocalSourceAddr("./beta"),
						Version:    nil,
					},
					"empty": {
						LocalName: "empty",
						Version:   nil,
					},
				},
			},
			want: []moduleCall{
				{
					Name:             "beta",
					SourceAddr:       "./beta",
					Version:          "",
					SourceType:       "local",
					DocsLink:         "",
					DependentModules: []moduleCall{},
				},
				{
					Name:             "ec2_instances",
					SourceAddr:       "terraform-aws-modules/ec2-instance/aws",
					Version:          "2.12.0",
					SourceType:       "tfregistry",
					DocsLink:         "https://registry.terraform.io/modules/terraform-aws-modules/ec2-instance/aws/latest?utm_content=workspace%2FexecuteCommand%2Fmodule.calls&utm_source=terraform-ls",
					DependentModules: []moduleCall{},
				},
				{
					Name:             "eks",
					SourceAddr:       "terraform-aws-modules/eks/aws",
					Version:          "17.20.0",
					SourceType:       "tfregistry",
					DocsLink:         "https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/latest?utm_content=workspace%2FexecuteCommand%2Fmodule.calls&utm_source=terraform-ls",
					DependentModules: []moduleCall{},
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
			ctx := context.Background()
			h := &CmdHandler{}
			got := h.parseModuleRecords(ctx, tt.moduleCalls)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("module mismatch: %s", diff)
			}
		})
	}
}

// With the release of Terraform 1.1.0 module source addresses are now stored normalized
func Test_parseModuleRecords_v1_1(t *testing.T) {
	tests := []struct {
		name        string
		moduleCalls tfmod.ModuleCalls
		want        []moduleCall
	}{
		{
			name: "detects terraform module types",
			moduleCalls: tfmod.ModuleCalls{
				Installed: map[string]tfmod.InstalledModuleCall{},
				Declared: map[string]tfmod.DeclaredModuleCall{
					"ec2_instances": {
						LocalName:  "ec2_instances",
						SourceAddr: tfaddr.MustParseModuleSource("registry.terraform.io/terraform-aws-modules/ec2-instance/aws"),
						Version:    version.MustConstraints(version.NewConstraint("2.12.0")),
					},
				},
			},
			want: []moduleCall{
				{
					Name:             "ec2_instances",
					SourceAddr:       "terraform-aws-modules/ec2-instance/aws",
					Version:          "2.12.0",
					SourceType:       "tfregistry",
					DocsLink:         "https://registry.terraform.io/modules/terraform-aws-modules/ec2-instance/aws/latest?utm_content=workspace%2FexecuteCommand%2Fmodule.calls&utm_source=terraform-ls",
					DependentModules: []moduleCall{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := &CmdHandler{}
			got := h.parseModuleRecords(ctx, tt.moduleCalls)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("module mismatch: %s", diff)
			}
		})
	}
}
