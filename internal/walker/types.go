// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package walker

import "github.com/hashicorp/terraform-ls/internal/document"

type DocumentStore interface {
	HasOpenDocuments(dirHandle document.DirHandle) (bool, error)
}
