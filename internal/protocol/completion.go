package protocol

import "github.com/hashicorp/hcl-lang/lang"

type CompletionItemR struct {
	CompletionItem

	ResolveHook *lang.ResolveHook `json:"data,omitempty"`
}
