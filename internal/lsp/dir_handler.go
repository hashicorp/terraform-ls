// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

type DirHandler interface {
	Dir() string
	URI() string
}
