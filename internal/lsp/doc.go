// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

/*
Package lsp provides functions to convert generic interfaces
to LSP-specific types.

This helps keeping individual packages independent of LSP
types which effectively represent 3rd party dependency
and centralizes all such conversion logic in one place.

It also enables consistency in how we convert data in both directions.
*/
