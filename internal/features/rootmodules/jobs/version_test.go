// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/stretchr/testify/mock"
	"github.com/zclconf/go-cty/cty"
)

// Test that the builtin provider schema obtained from a root module
// wins over the (possibly older) preloaded one for modules it calls
func TestGetTerraformVersion_builtinProvider(t *testing.T) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	rootPath := "workspace"
	err = rs.Add(rootPath)
	if err != nil {
		t.Fatal(err)
	}

	builtinAddr := tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform")

	err = gs.ProviderSchemas.AddPreloadedSchema(builtinAddr, testVersion(t, "1.15.8"), &tfschema.ProviderSchema{
		Provider: &schema.BodySchema{
			Description: lang.PlainText("preload: builtin/terraform 1.15.8"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = exec.WithExecutorOpts(ctx, &exec.ExecutorOpts{
		ExecPath: "mock",
	})
	ctx = exec.WithExecutorFactory(ctx, exec.NewMockExecutor(&exec.TerraformMockCalls{
		AnyWorkDir: []*mock.Call{
			{
				Method:        "Version",
				Repeatability: 1,
				Arguments: []interface{}{
					mock.AnythingOfType(""),
				},
				ReturnArguments: []interface{}{
					testVersion(t, "1.16.2"),
					map[string]*version.Version{
						"registry.terraform.io/hashicorp/random": testVersion(t, "3.9.1"),
					},
					nil,
				},
			},
			{
				Method:        "ProviderSchemas",
				Repeatability: 1,
				Arguments: []interface{}{
					mock.AnythingOfType(""),
				},
				ReturnArguments: []interface{}{
					&tfjson.ProviderSchemas{
						FormatVersion: "1.0",
						Schemas: map[string]*tfjson.ProviderSchema{
							builtinAddr.String(): {
								ConfigSchema: &tfjson.Schema{
									Block: &tfjson.SchemaBlock{
										Description: "local: builtin/terraform 1.16.2",
									},
								},
								ResourceSchemas: map[string]*tfjson.Schema{
									"terraform_data": {
										Block: &tfjson.SchemaBlock{
											NestedBlocks: map[string]*tfjson.SchemaBlockType{
												"store": {
													NestingMode: tfjson.SchemaNestingModeSingle,
													Block: &tfjson.SchemaBlock{
														Attributes: map[string]*tfjson.SchemaAttribute{
															"input": {
																AttributeType: cty.DynamicPseudoType,
																Optional:      true,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					nil,
				},
			},
		},
	}))

	err = GetTerraformVersion(ctx, rs, rootPath)
	if err != nil {
		t.Fatal(err)
	}
	err = ObtainSchema(ctx, rs, gs.ProviderSchemas, rootPath)
	if err != nil {
		t.Fatal(err)
	}

	modPath := filepath.Join(rootPath, "modules", "module1")
	ps, err := gs.ProviderSchemas.ProviderSchema(modPath, builtinAddr, version.Constraints{})
	if err != nil {
		t.Fatal(err)
	}
	if ps == nil {
		t.Fatal("no schema found")
	}

	expectedDescription := "local: builtin/terraform 1.16.2"
	if ps.Provider.Description.Value != expectedDescription {
		t.Fatalf("description doesn't match. expected: %q, got: %q",
			expectedDescription, ps.Provider.Description.Value)
	}
	if _, ok := ps.Resources["terraform_data"].Blocks["store"]; !ok {
		t.Fatal("expected store block in terraform_data schema")
	}
}
