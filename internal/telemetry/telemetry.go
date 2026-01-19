// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package telemetry

import (
	"context"
	"fmt"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

type Telemetry struct {
	version  int
	notifier Notifier
}

type Notifier interface {
	Notify(ctx context.Context, method string, params interface{}) error
}

func NewSender(version int, notifier Notifier) (*Telemetry, error) {
	if version != lsp.TelemetryFormatVersion {
		return nil, fmt.Errorf("unsupported telemetry format version: %d", version)
	}

	return &Telemetry{
		version:  version,
		notifier: notifier,
	}, nil
}

func (t *Telemetry) SendEvent(ctx context.Context, name string, properties map[string]interface{}) {
	t.notifier.Notify(ctx, "telemetry/event", lsp.TelemetryEvent{
		Version:    t.version,
		Name:       name,
		Properties: properties,
	})
}

func IsPublicProvider(addr tfaddr.Provider) bool {
	if addr.Hostname == tfaddr.DefaultProviderRegistryHost {
		return true
	}
	if addr.IsLegacy() || addr.IsBuiltIn() {
		return true
	}
	return false
}
