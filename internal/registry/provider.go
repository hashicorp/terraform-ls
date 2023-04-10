// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type pagination struct {
	NextPage int `json:"next-page"`
}

type meta struct {
	Pagination pagination `json:"pagination"`
}

type registryResponse struct {
	Data []Provider `json:"data"`
	Meta meta       `json:"meta"`
}

type ProviderAttributes struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Provider struct {
	ID         string             `json:"id"`
	Attributes ProviderAttributes `json:"attributes"`
}

func (c Client) ListProviders(tier string) ([]Provider, error) {
	var providers []Provider
	page := 1
	for page > 0 {
		url := fmt.Sprintf("%s/v2/providers?page[size]=%d&filter[tier]=%s&page[number]=%d",
			c.BaseURL, c.ProviderPageSize, tier, page)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			return nil, fmt.Errorf("unexpected response: %s: %s", resp.Status, string(bodyBytes))
		}

		var response registryResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, fmt.Errorf("unable to decode response: %w", err)
		}
		providers = append(providers, response.Data...)
		page = response.Meta.Pagination.NextPage
	}
	return providers, nil
}

type ProviderVersionResponse struct {
	Data     ProviderVersionData `json:"data"`
	Included []Included          `json:"included"`
}

type Included struct {
	Type       string             `json:"type"`
	Attributes IncludedAttributes `json:"attributes"`
}

type IncludedAttributes struct {
	Arch string `json:"arch"`
	Os   string `json:"os"`
}

type ProviderVersionData struct {
	Attributes ProviderVersionAttributes `json:"attributes"`
}

type ProviderVersionAttributes struct {
	Version string `json:"version"`
}

func (c Client) GetLatestProviderVersion(id string) (*ProviderVersionResponse, error) {
	url := fmt.Sprintf("%s/v2/providers/%s/provider-versions/latest?include=provider-platforms",
		c.BaseURL, id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		return nil, fmt.Errorf("unexpected response %s: %s", resp.Status, string(bodyBytes))
	}

	var response ProviderVersionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
