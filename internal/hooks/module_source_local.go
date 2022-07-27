package hooks

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/zclconf/go-cty/cty"
)

func (h *Hooks) LocalModuleSources(ctx context.Context, value cty.Value) ([]decoder.Candidate, error) {
	candidates := make([]decoder.Candidate, 0)

	// Obtain indexed modules via h.modStore.List()
	// TODO filter modules inside .terraform
	// TODO build candidates

	return candidates, nil
}
