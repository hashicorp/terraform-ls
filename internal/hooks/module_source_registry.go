package hooks

import (
	"context"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/zclconf/go-cty/cty"
)

type RegistryModule struct {
	FullName    string `json:"full-name"`
	Description string `json:"description"`
}

const algoliaModuleIndex = "tf-registry:prod:modules"

func (h *Hooks) fetchModulesFromAlgolia(ctx context.Context, term string) ([]RegistryModule, error) {
	modules := make([]RegistryModule, 0)

	index := h.AlgoliaClient.InitIndex(algoliaModuleIndex)
	params := []interface{}{
		ctx, // transport.Request will magically extract the context from here
		opt.AttributesToRetrieve("full-name", "description"),
		opt.HitsPerPage(10),
	}

	res, err := index.Search(term, params...)
	if err != nil {
		return modules, err
	}

	err = res.UnmarshalHits(&modules)
	if err != nil {
		return modules, err

	}

	return modules, nil
}

func (h *Hooks) RegistryModuleSources(ctx context.Context, value cty.Value) ([]decoder.Candidate, error) {
	candidates := make([]decoder.Candidate, 0)
	prefix := value.AsString()

	if isModuleSourceLocal(prefix) {
		// We're dealing with a local module source here, no need to search the registry
		return candidates, nil
	}

	if h.AlgoliaClient == nil {
		return candidates, nil
	}

	modules, err := h.fetchModulesFromAlgolia(ctx, prefix)
	if err != nil {
		return candidates, err
	}

	for _, mod := range modules {
		c := decoder.ExpressionCompletionCandidate(decoder.ExpressionCandidate{
			Value:       cty.StringVal(mod.FullName),
			Detail:      "registry",
			Description: lang.PlainText(mod.Description),
		})
		candidates = append(candidates, c)
	}

	return candidates, nil
}

var moduleSourceLocalPrefixes = []string{
	"./",
	"../",
	".\\",
	"..\\",
}

func isModuleSourceLocal(raw string) bool {
	for _, prefix := range moduleSourceLocalPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}
