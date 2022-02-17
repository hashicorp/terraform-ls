package document

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-ls/internal/uri"
)

// Handle represents a document location
//
// This may be received via LSP from the client (as URI)
// or constructed from a file path on OS FS.
type Handle struct {
	Dir      DirHandle
	Filename string
}

// HandleFromURI creates a Handle from a given URI.
//
// docURI is expected to be a document URI (rather than dir).
// It is however outside the scope of the function to verify
// this is actually the case or whether the file exists.
func HandleFromURI(docUri string) Handle {
	path := uri.MustPathFromURI(docUri)

	filename := filepath.Base(path)
	dirUri := strings.TrimSuffix(docUri, "/"+filename)

	return Handle{
		Dir:      DirHandle{URI: dirUri},
		Filename: filename,
	}
}

// HandleFromPath creates a Handle from a given path.
//
// docPath is expected to be a document path (rather than dir).
// It is however outside the scope of the function to verify
// this is actually the case or whether the file exists.
func HandleFromPath(docPath string) Handle {
	docUri := uri.FromPath(docPath)

	filename := filepath.Base(docPath)
	dirUri := strings.TrimSuffix(docUri, "/"+filename)

	return Handle{
		Dir:      DirHandle{URI: dirUri},
		Filename: filename,
	}
}

func (h Handle) FullPath() string {
	return filepath.Join(h.Dir.Path(), h.Filename)
}

func (h Handle) FullURI() string {
	return h.Dir.URI + "/" + h.Filename
}
