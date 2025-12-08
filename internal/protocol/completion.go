// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package protocol

import "github.com/hashicorp/hcl-lang/lang"

type CompletionItemWithResolveHook struct {
	CompletionItem

	ResolveHook *lang.ResolveHook `json:"data,omitempty"`
}
