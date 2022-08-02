package hooks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/registry"
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

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	regClient.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	h := &Hooks{
		ModStore:       s.Modules,
		RegistryClient: regClient,
	}

	modules := []string{
		tmpDir,
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
		filepath.Join(tmpDir, "..", "gamma"),
		filepath.Join(".terraform", "modules", "web_server_sg"),
	}

	for _, mod := range modules {
		err := s.Modules.Add(mod)
		if err != nil {
			t.Fatal(err)
		}
	}

	expectedCandidates := []decoder.Candidate{
		{
			Label:         "\"./alpha\"",
			Detail:        "local",
			Kind:          lang.StringCandidateKind,
			RawInsertText: "\"./alpha\"",
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
