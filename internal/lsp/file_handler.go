// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func HandleFromDocumentURI(docUri lsp.DocumentURI) document.Handle {
	return document.HandleFromURI(string(docUri))
}

func DirHandleFromDirURI(dirUri lsp.DocumentURI) document.DirHandle {
	return document.DirHandleFromURI(string(dirUri))
}
