package lsp

import (
	"fmt"

	lsp "github.com/sourcegraph/go-lsp"
)

type InvalidLspPosErr struct {
	Pos lsp.Position
}

func (e *InvalidLspPosErr) Error() string {
	return fmt.Sprintf("invalid position: %s", e.Pos)
}
