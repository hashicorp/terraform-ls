// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

func IsNoBlockFoundErr(err error) bool {
	_, ok := err.(*NoBlockFoundErr)
	return ok
}

type NoTokenFoundErr struct {
	AtPos hcllib.Pos
}

func (e *NoTokenFoundErr) Error() string {
	return fmt.Sprintf("no token found at %#v", e.AtPos)
}
