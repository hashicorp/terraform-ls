package hcl

import (
	"fmt"

	hcllib "github.com/hashicorp/hcl/v2"
)

type InvalidHclPosErr struct {
	Pos     hcllib.Pos
	InRange hcllib.Range
}

func (e *InvalidHclPosErr) Error() string {
	return fmt.Sprintf("invalid position: %#v in %s", e.Pos, e.InRange.String())
}

type NoBlockFoundErr struct {
	AtPos hcllib.Pos
}

func (e *NoBlockFoundErr) Error() string {
	return fmt.Sprintf("no block found at %#v", e.AtPos)
}
