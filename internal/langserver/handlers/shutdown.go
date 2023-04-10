// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
)

func Shutdown(ctx context.Context, _ interface{}) error {
	return nil
}
