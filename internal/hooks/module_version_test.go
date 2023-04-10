// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hooks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/zclconf/go-cty/cty"
)

var moduleVersionsMockResponse = `{
	"modules": [
	  {
		"source": "terraform-aws-modules/vpc/aws",
		"versions": [
		  {
			"version": "0.0.1"
		  },
		  {
			"version": "2.0.24"
		  },
		  {
			"version": "1.33.7"
		  }
		]
	  }
	]
  }`

func TestHooks_RegistryModuleVersions(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ctx = decoder.WithPath(ctx, lang.Path{
		Path:       tmpDir,
		LanguageID: "terraform",
	})
	ctx = decoder.WithPos(ctx, hcl.Pos{
		Line:   2,
		Column: 5,
		Byte:   5,
	})
	ctx = decoder.WithFilename(ctx, "main.tf")
	ctx = decoder.WithMaxCandidates(ctx, 3)
	s, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/terraform-aws-modules/vpc/aws/versions" {
			w.Write([]byte(moduleVersionsMockResponse))
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	regClient.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	h := &Hooks{
		ModStore:       s.Modules,
		RegistryClient: regClient,
	}

	err = s.Modules.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	metadata := &tfmod.Meta{
		Path: tmpDir,
		ModuleCalls: map[string]tfmod.DeclaredModuleCall{
			"vpc": {
				LocalName:  "vpc",
				SourceAddr: tfaddr.MustParseModuleSource("registry.terraform.io/terraform-aws-modules/vpc/aws"),
				RangePtr: &hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 1},
					End:      hcl.Pos{Line: 4, Column: 2, Byte: 20},
				},
			},
		},
	}
	err = s.Modules.UpdateMetadata(tmpDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedCandidates := []decoder.Candidate{
		{
			Label:         `"2.0.24"`,
			Kind:          lang.StringCandidateKind,
			RawInsertText: `"2.0.24"`,
			SortText:      "  0",
		},
		{
			Label:         `"1.33.7"`,
			Kind:          lang.StringCandidateKind,
			RawInsertText: `"1.33.7"`,
			SortText:      "  1",
		},
		{
			Label:         `"0.0.1"`,
			Kind:          lang.StringCandidateKind,
			RawInsertText: `"0.0.1"`,
			SortText:      "  2",
		},
	}

	candidates, _ := h.RegistryModuleVersions(ctx, cty.StringVal(""))
	if diff := cmp.Diff(expectedCandidates, candidates); diff != "" {
		t.Fatalf("mismatched candidates: %s", diff)
	}
}
