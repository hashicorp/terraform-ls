// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"github.com/hashicorp/terraform-ls/internal/uri"
)

// DirHandle represents a directory location
//
// This may be received via LSP from the client (as URI)
// or constructed from a file path on OS FS.
type DirHandle struct {
	URI string
}

func (dh DirHandle) Path() string {
	return uri.MustPathFromURI(dh.URI)
}

// DirHandleFromPath creates a DirHandle from a given path.
//
// dirPath is expected to be a directory path (rather than document).
// It is however outside the scope of the function to verify
// this is actually the case or whether the directory exists.
func DirHandleFromPath(dirPath string) DirHandle {
	return DirHandle{
		URI: uri.FromPath(dirPath),
	}
}

// DirHandleFromURI creates a DirHandle from a given URI.
//
// dirUri is expected to be a directory URI (rather than document).
// It is however outside the scope of the function to verify
// this is actually the case or whether the directory exists.
func DirHandleFromURI(dirUri string) DirHandle {
	// Normalize the raw URI to account for any escaping differences
	dirUri = uri.MustParseURI(dirUri)

	return DirHandle{
		URI: dirUri,
	}
}
