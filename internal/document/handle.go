package document

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-ls/internal/uri"
)

type Handle struct {
	Dir      DirHandle
	Filename string
}

func HandleFromURI(docUri string) Handle {
	path := uri.MustPathFromURI(docUri)

	filename := filepath.Base(path)
	dirUri := strings.TrimSuffix(docUri, "/"+filename)

	return Handle{
		Dir:      DirHandle{URI: dirUri},
		Filename: filename,
	}
}

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
