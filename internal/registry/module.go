// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"sort"

	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ModuleResponse struct {
	Version string     `json:"version"`
	Root    ModuleRoot `json:"root"`
}

type ModuleRoot struct {
	Inputs  []Input  `json:"inputs"`
	Outputs []Output `json:"outputs"`
}

type Input struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
}

type Output struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ModuleVersionsResponse struct {
	Modules []ModuleVersionsEntry `json:"modules"`
}

type ModuleVersionsEntry struct {
	Versions []ModuleVersion `json:"versions"`
}

type ModuleVersion struct {
	Version string `json:"version"`
}

type ClientError struct {
	StatusCode int
	Body       string
}

func (rce ClientError) Error() string {
	return fmt.Sprintf("%d: %s", rce.StatusCode, rce.Body)
}

func (c Client) GetModuleData(ctx context.Context, addr tfaddr.Module, cons version.Constraints) (*ModuleResponse, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "registry:GetModuleData")
	defer span.End()
	var response ModuleResponse

	v, err := c.GetMatchingModuleVersion(ctx, addr, cons)
	if err != nil {
		return nil, err
	}

	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx, otelhttptrace.WithoutSubSpans()))

	url := fmt.Sprintf("%s/v1/modules/%s/%s/%s/%s", c.BaseURL,
		addr.Package.Namespace,
		addr.Package.Name,
		addr.Package.TargetSystem,
		v.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, ClientError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c Client) GetMatchingModuleVersion(ctx context.Context, addr tfaddr.Module, con version.Constraints) (*version.Version, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "registry:GetMatchingModuleVersion")
	defer span.End()
	foundVersions, err := c.GetModuleVersions(ctx, addr)
	if err != nil {
		return nil, err
	}

	for _, fv := range foundVersions {
		if con.Check(fv) {
			return fv, nil
		}
	}

	return nil, fmt.Errorf("no suitable version found for %q %q", addr, con)
}

func (c Client) GetModuleVersions(ctx context.Context, addr tfaddr.Module) (version.Collection, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "registry:GetModuleVersions")
	defer span.End()

	url := fmt.Sprintf("%s/v1/modules/%s/%s/%s/versions", c.BaseURL,
		addr.Package.Namespace,
		addr.Package.Name,
		addr.Package.TargetSystem)

	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx, otelhttptrace.WithoutSubSpans()))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, ClientError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	_, decodeSpan := otel.Tracer(tracerName).Start(ctx, "registry:GetModuleVersions:decodeJson")
	var response ModuleVersionsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	decodeSpan.End()

	var foundVersions version.Collection
	for _, module := range response.Modules {
		for _, entry := range module.Versions {
			ver, err := version.NewVersion(entry.Version)
			if err == nil {
				foundVersions = append(foundVersions, ver)
			}
		}
	}
	span.AddEvent("registry:foundModuleVersions",
		trace.WithAttributes(attribute.KeyValue{
			Key:   attribute.Key("moduleVersionCount"),
			Value: attribute.IntValue(len(foundVersions)),
		}))

	sort.Sort(sort.Reverse(foundVersions))

	return foundVersions, nil
}
