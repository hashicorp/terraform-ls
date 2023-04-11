// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protocol

import "github.com/hashicorp/hcl-lang/lang"

type CompletionItemWithResolveHook struct {
	CompletionItem

	ResolveHook *lang.ResolveHook `json:"data,omitempty"`
}
