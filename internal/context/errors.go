// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package context

import "fmt"

type MissingContextErr struct {
	CtxKey *contextKey
}

func (e *MissingContextErr) Error() string {
	return fmt.Sprintf("missing context: %s", e.CtxKey)
}
