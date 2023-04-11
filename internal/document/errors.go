// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"fmt"
)

type InvalidPosErr struct {
	Pos Pos
}

func (e *InvalidPosErr) Error() string {
	return fmt.Sprintf("invalid position: %s", e.Pos)
}

type DocumentNotFound struct {
	URI string
}

func (e *DocumentNotFound) Error() string {
	msg := "document not found"
	if e.URI != "" {
		return fmt.Sprintf("%s: %s", e.URI, msg)
	}

	return msg
}

func (e *DocumentNotFound) Is(err error) bool {
	_, ok := err.(*DocumentNotFound)
	return ok
}
