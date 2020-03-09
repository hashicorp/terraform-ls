package filesystem

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/sourcegraph/go-lsp"
)

type NoBlockFoundErr struct {
	AtPos hcl.Pos
}

func (e *NoBlockFoundErr) Error() string {
	return fmt.Sprintf("no block found at %#v", e.AtPos)
}

type InvalidPosErr struct {
	Pos lsp.Position
}

func (e *InvalidPosErr) Error() string {
	return fmt.Sprintf("invalid position: %s", e.Pos)
}
