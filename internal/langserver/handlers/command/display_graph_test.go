// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"
	"testing"

	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func Test_GetNodes(t *testing.T) {
	tests := []struct {
		name            string
		pathDecoder     *decoder.PathDecoder
		path            lang.Path
		expectedNodes   []node
		expectedNodeMap map[string]int
	}{
		{
			name: "single file",
			pathDecoder: createTestPathDecoder(t, map[string]string{
				"main.tf": `
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1d0"
  instance_type = "t2.micro"
}

variable "region" {
  type = string
}
`,
			}, &schema.BodySchema{
				Blocks: map[string]*schema.BlockSchema{
					"resource": {
						Labels: []*schema.LabelSchema{
							{Name: "type"},
							{Name: "name"},
						},
						Body: &schema.BodySchema{},
					},
					"variable": {
						Labels: []*schema.LabelSchema{
							{Name: "name"},
						},
						Body: &schema.BodySchema{},
					},
				},
			}),
			path: lang.Path{Path: "/test", LanguageID: "terraform"},
			expectedNodes: []node{
				{
					ID: 0,
					Location: lsp.Location{
						URI: lsp.DocumentURI(uri.FromPath("/test/main.tf")),
						Range: ilsp.HCLRangeToLSP(hcl.Range{
							Filename: "main.tf",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 34, Byte: 33},
						}),
					},
					Type:   "resource",
					Labels: []string{"aws_instance", "example"},
				},
				{
					ID: 1,
					Location: lsp.Location{
						URI: lsp.DocumentURI(uri.FromPath("/test/main.tf")),
						Range: ilsp.HCLRangeToLSP(hcl.Range{
							Filename: "main.tf",
							Start:    hcl.Pos{Line: 7, Column: 1, Byte: 90},
							End:      hcl.Pos{Line: 7, Column: 18, Byte: 107},
						}),
					},
					Type:   "variable",
					Labels: []string{"region"},
				},
			},
			expectedNodeMap: map[string]int{
				locationKey(lsp.Location{
					URI: lsp.DocumentURI(uri.FromPath("/test/main.tf")),
					Range: ilsp.HCLRangeToLSP(hcl.Range{
						Filename: "main.tf",
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 34, Byte: 33},
					}),
				}): 0,
				locationKey(lsp.Location{
					URI: lsp.DocumentURI(uri.FromPath("/test/main.tf")),
					Range: ilsp.HCLRangeToLSP(hcl.Range{
						Filename: "main.tf",
						Start:    hcl.Pos{Line: 7, Column: 1, Byte: 90},
						End:      hcl.Pos{Line: 7, Column: 18, Byte: 107},
					}),
				}): 1,
			},
		},
		{
			name: "multiple files",
			pathDecoder: createTestPathDecoder(t, map[string]string{
				"main.tf": `
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1d0"
  instance_type = "t2.micro"
}
`,
				"variables.tf": `
variable "region" {
  type = string
}

variable "instance_type" {
  type = string
}
`,
			}, &schema.BodySchema{
				Blocks: map[string]*schema.BlockSchema{
					"resource": {
						Labels: []*schema.LabelSchema{
							{Name: "type"},
							{Name: "name"},
						},
						Body: &schema.BodySchema{},
					},
					"variable": {
						Labels: []*schema.LabelSchema{
							{Name: "name"},
						},
						Body: &schema.BodySchema{},
					},
				},
			}),
			path: lang.Path{Path: "/test", LanguageID: "terraform"},
			expectedNodes: []node{
				{
					ID: 0,
					Location: lsp.Location{
						URI: lsp.DocumentURI(uri.FromPath("/test/main.tf")),
						Range: ilsp.HCLRangeToLSP(hcl.Range{
							Filename: "main.tf",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 34, Byte: 33},
						}),
					},
					Type:   "resource",
					Labels: []string{"aws_instance", "example"},
				},
				{
					ID: 1,
					Location: lsp.Location{
						URI: lsp.DocumentURI(uri.FromPath("/test/variables.tf")),
						Range: ilsp.HCLRangeToLSP(hcl.Range{
							Filename: "variables.tf",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 18, Byte: 17},
						}),
					},
					Type:   "variable",
					Labels: []string{"region"},
				},
				{
					ID: 2,
					Location: lsp.Location{
						URI: lsp.DocumentURI(uri.FromPath("/test/variables.tf")),
						Range: ilsp.HCLRangeToLSP(hcl.Range{
							Filename: "variables.tf",
							Start:    hcl.Pos{Line: 6, Column: 1, Byte: 34},
							End:      hcl.Pos{Line: 6, Column: 25, Byte: 58},
						}),
					},
					Type:   "variable",
					Labels: []string{"instance_type"},
				},
			},
			expectedNodeMap: map[string]int{
				locationKey(lsp.Location{
					URI: lsp.DocumentURI(uri.FromPath("/test/main.tf")),
					Range: ilsp.HCLRangeToLSP(hcl.Range{
						Filename: "main.tf",
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 34, Byte: 33},
					}),
				}): 0,
				locationKey(lsp.Location{
					URI: lsp.DocumentURI(uri.FromPath("/test/variables.tf")),
					Range: ilsp.HCLRangeToLSP(hcl.Range{
						Filename: "variables.tf",
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
						End:      hcl.Pos{Line: 2, Column: 18, Byte: 17},
					}),
				}): 1,
				locationKey(lsp.Location{
					URI: lsp.DocumentURI(uri.FromPath("/test/variables.tf")),
					Range: ilsp.HCLRangeToLSP(hcl.Range{
						Filename: "variables.tf",
						Start:    hcl.Pos{Line: 6, Column: 1, Byte: 34},
						End:      hcl.Pos{Line: 6, Column: 25, Byte: 58},
					}),
				}): 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, nodeMap, err := getNodes(tt.pathDecoder, tt.path)
			if err != nil {
				t.Fatalf("getNodes() error = %v", err)
			}

			if diff := cmp.Diff(tt.expectedNodes, nodes); diff != "" {
				t.Errorf("nodes mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.expectedNodeMap, nodeMap); diff != "" {
				t.Errorf("nodeMap mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_GetEdges(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		schema        *schema.BodySchema
		expectedEdges []edge
	}{
		{
			name: "variable and local referenced in output",
			files: map[string]string{
				"main.tf": `
variable "region" {
  type = string
}

locals {
  region = var.region
}

output "region" {
  value = local.region
}
`,
			},
			schema: &schema.BodySchema{
				Blocks: map[string]*schema.BlockSchema{
					"variable": {
						Address: &schema.BlockAddrSchema{
							Steps: []schema.AddrStep{
								schema.StaticStep{Name: "var"},
								schema.LabelStep{Index: 0},
							},
							AsReference: true,
							ScopeId:     lang.ScopeId("variable"),
						},
						Labels: []*schema.LabelSchema{
							{Name: "name"},
						},
						Body: &schema.BodySchema{
							Attributes: map[string]*schema.AttributeSchema{
								"type": {
									Constraint: schema.TypeDeclaration{},
									IsOptional: true,
								},
							},
						},
					},
					"locals": {
						Body: &schema.BodySchema{
							Attributes: map[string]*schema.AttributeSchema{
								"region": {
									Address: &schema.AttributeAddrSchema{
										Steps: []schema.AddrStep{
											schema.StaticStep{Name: "local"},
											schema.AttrNameStep{},
										},
										ScopeId:     lang.ScopeId("local"),
										AsExprType:  true,
										AsReference: true,
									},
									Constraint: schema.Reference{
										OfScopeId: lang.ScopeId("variable"),
									},
								},
							},
						},
					},
					"output": {
						Labels: []*schema.LabelSchema{
							{Name: "name"},
						},
						Body: &schema.BodySchema{
							Attributes: map[string]*schema.AttributeSchema{
								"value": {
									Constraint: schema.Reference{
										OfScopeId: lang.ScopeId("local"),
									},
									IsOptional: true,
								},
							},
						},
					},
				},
			},
			expectedEdges: []edge{
				{
					From: 1,
					To:   2,
				},
				{
					From: 0,
					To:   1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathDecoder, d, path := createTestDecoder(t, tt.files, tt.schema)

			_, nodeMap, err := getNodes(pathDecoder, path)
			if err != nil {
				t.Fatalf("getNodes() error = %v", err)
			}

			edges, err := getEdges(pathDecoder, path, d, nodeMap)
			if err != nil {
				t.Fatalf("getEdges() error = %v", err)
			}

			if diff := cmp.Diff(tt.expectedEdges, edges); diff != "" {
				t.Errorf("edges mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func createTestDecoder(t *testing.T, files map[string]string, schema *schema.BodySchema) (*decoder.PathDecoder, *decoder.Decoder, lang.Path) {
	pathCtx := &decoder.PathContext{
		Schema: schema,
		Files:  make(map[string]*hcl.File),
	}

	p := hclparse.NewParser()
	dirPath := "/test"
	for filename, content := range files {
		file, diags := p.ParseHCL([]byte(content), filename)
		if len(diags) > 0 {
			t.Fatalf("failed to parse HCL for %s: %v", filename, diags)
		}
		pathCtx.Files[filepath.Join(dirPath, filename)] = file
	}
	dirs := map[string]*decoder.PathContext{
		dirPath: pathCtx,
	}

	d := decoder.NewDecoder(&testPathReader{
		paths: dirs,
	})
	d.SetContext(decoder.NewDecoderContext())

	path := lang.Path{Path: dirPath, LanguageID: "terraform"}

	// First create a temporary PathDecoder to collect reference targets and origins
	tempPathDecoder, err := d.Path(path)
	if err != nil {
		t.Fatal(err)
	}
	refTargets, err := tempPathDecoder.CollectReferenceTargets()
	if err != nil {
		t.Fatal(err)
	}
	refOrigins, err := tempPathDecoder.CollectReferenceOrigins()
	if err != nil {
		t.Fatal(err)
	}

	// Set the collected reference targets and origins on the path context
	dirs[dirPath].ReferenceTargets = refTargets
	dirs[dirPath].ReferenceOrigins = refOrigins

	// Now create the final PathDecoder with the populated reference targets
	pathDecoder, err := d.Path(path)
	if err != nil {
		t.Fatal(err)
	}

	return pathDecoder, d, path
}

func createTestPathDecoder(t *testing.T, files map[string]string, schema *schema.BodySchema) *decoder.PathDecoder {
	pathDecoder, _, _ := createTestDecoder(t, files, schema)
	return pathDecoder
}

type testPathReader struct {
	paths map[string]*decoder.PathContext
}

func (r *testPathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, len(r.paths))

	i := 0
	for path := range r.paths {
		paths[i] = lang.Path{Path: path, LanguageID: "terraform"}
		i++
	}

	return paths
}

func (r *testPathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	if ctx, ok := r.paths[path.Path]; ok {
		return ctx, nil
	}

	return nil, fmt.Errorf("path not found: %q", path.Path)
}
