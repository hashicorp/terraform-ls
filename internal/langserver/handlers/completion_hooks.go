package handlers

import (
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/terraform-ls/internal/hooks"
)

func (s *service) AppendCompletionHooks(ctx decoder.DecoderContext) {
	h := hooks.Hooks{
		ModStore:       s.modStore,
		RegistryClient: s.registryClient,
	}

	ctx.CompletionHooks["CompleteLocalModuleSources"] = h.LocalModuleSources

}
