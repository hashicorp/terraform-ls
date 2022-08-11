package hooks

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/zclconf/go-cty/cty"
)

func TestHooks_LocalModuleSources(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ctx = decoder.WithPath(ctx, lang.Path{
		Path:       tmpDir,
		LanguageID: "terraform",
	})
	s, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	h := &Hooks{
		ModStore: s.Modules,
	}

	modules := []string{
		tmpDir,
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
		filepath.Join(tmpDir, "..", "gamma"),
		filepath.Join(".terraform", "modules", "web_server_sg"),
		filepath.Join(tmpDir, "any.terraformany"),
		filepath.Join(tmpDir, "any.terraform"),
		filepath.Join(tmpDir, ".terraformany"),
	}

	for _, mod := range modules {
		err := s.Modules.Add(mod)
		if err != nil {
			t.Fatal(err)
		}
	}

	expectedCandidates := []decoder.Candidate{
		{
			Label:         "\"./.terraformany\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"./.terraformany\"",
		},
		{
			Label:         "\"./alpha\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"./alpha\"",
		},
		{
			Label:         "\"./any.terraform\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"./any.terraform\"",
		},
		{
			Label:         "\"./any.terraformany\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"./any.terraformany\"",
		},
		{
			Label:         "\"./beta\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"./beta\"",
		},
		{
			Label:         "\"../gamma\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"../gamma\"",
		},
	}

	candidates, _ := h.LocalModuleSources(ctx, cty.StringVal(""))
	if diff := cmp.Diff(expectedCandidates, candidates); diff != "" {
		t.Fatalf("mismatched candidates: %s", diff)
	}
}
