// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package telemetry

import (
	"context"

	"github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"go.opentelemetry.io/otel/attribute"
)

func InitHoneycomb(apiKey string, attributeKvs []attribute.KeyValue) (shutdownFunc, error) {
	attributes := make(map[string]string)
	for _, kvPair := range attributeKvs {
		// TODO: We can revisit this to send the attributes as the right type
		// once https://github.com/honeycombio/otel-config-go/pull/48 is merged.
		attributes[string(kvPair.Key)] = kvPair.Value.Emit()
	}

	bsp := honeycomb.NewBaggageSpanProcessor()
	f, err := otelconfig.ConfigureOpenTelemetry(
		otelconfig.WithSpanProcessor(bsp),
		otelconfig.WithResourceAttributes(attributes),
		otelconfig.WithLogLevel("debug"),
		otelconfig.WithExporterEndpoint("api.honeycomb.io:443"),
		otelconfig.WithHeaders(map[string]string{
			"x-honeycomb-team": apiKey,
		}),
	)

	return func(_ context.Context) error {
		f()
		return nil
	}, err
}
