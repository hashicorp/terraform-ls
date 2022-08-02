package handlers

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/terraform-ls/internal/hooks"
)

var AlgoliaAppID = ""
var AlgoliaAPIKey = ""

func (s *service) AppendCompletionHooks(decoderContext decoder.DecoderContext) {
	h := hooks.Hooks{
		ModStore:       s.modStore,
		RegistryClient: s.registryClient,
	}

	if AlgoliaAppID != "" && AlgoliaAPIKey != "" {
		h.AlgoliaClient = search.NewClient(AlgoliaAppID, AlgoliaAPIKey)
	}

	decoderContext.CompletionHooks["CompleteLocalModuleSources"] = h.LocalModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleSources"] = h.RegistryModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleVersions"] = h.RegistryModuleVersions
}
