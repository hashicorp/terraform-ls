// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package telemetry

import "context"

type Sender interface {
	SendEvent(ctx context.Context, name string, properties map[string]interface{})
}
