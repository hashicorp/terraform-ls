package document

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-ls/internal/uri"
)

type DirHandle struct {
	URI string
}

func (dh DirHandle) Path() string {
	return uri.MustPathFromURI(dh.URI)
}

func DirHandleFromPath(path string) DirHandle {
	path = strings.TrimSuffix(path, fmt.Sprintf("%c", os.PathSeparator))

	return DirHandle{
		URI: uri.FromPath(path),
	}
}

func DirHandleFromURI(dirUri string) DirHandle {
	// Dir URIs are usually without trailing separator already
	// but we double check anyway, so we deal with the same URI
	// regardless of language client differences
	dirUri = strings.TrimSuffix(string(dirUri), "/")

	return DirHandle{
		URI: dirUri,
	}
}
