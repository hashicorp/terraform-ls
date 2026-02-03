// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package lsp

type DirHandler interface {
	Dir() string
	URI() string
}
