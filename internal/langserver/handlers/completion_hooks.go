package handlers

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/hashicorp/hcl-lang/decoder"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/hooks"
)

func (s *service) AppendCompletionHooks(decoderContext decoder.DecoderContext) {
	h := hooks.Hooks{
		ModStore:       s.modStore,
		RegistryClient: s.registryClient,
	}

	credentials, ok := lsctx.AlgoliaCredentials(s.srvCtx)
	if ok {
		h.AlgoliaClient = search.NewClient(credentials.AlgoliaAppID, credentials.AlgoliaAPIKey)
	}

	decoderContext.CompletionHooks["CompleteLocalModuleSources"] = h.LocalModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleSources"] = h.RegistryModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleVersions"] = h.RegistryModuleVersions
}
