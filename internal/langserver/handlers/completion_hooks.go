// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/terraform-ls/internal/algolia"
	"github.com/hashicorp/terraform-ls/internal/hooks"
)

func (s *service) AppendCompletionHooks(decoderContext decoder.DecoderContext) {
	h := hooks.Hooks{
		ModStore:       s.recordStores.Modules,
		RegistryClient: s.registryClient,
		Logger:         s.logger,
	}

	credentials, ok := algolia.CredentialsFromContext(s.srvCtx)
	if ok {
		h.AlgoliaClient = search.NewClient(credentials.AppID, credentials.APIKey)
	}

	decoderContext.CompletionHooks["CompleteLocalModuleSources"] = h.LocalModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleSources"] = h.RegistryModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleVersions"] = h.RegistryModuleVersions
}
