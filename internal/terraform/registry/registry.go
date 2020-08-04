package registry

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
)

var registryProviders = make([]addrs.Provider, 0)
var err error
var o = &sync.Once{}

func GetProviders() (*[]addrs.Provider, error) {
	o.Do(func() {
		var result *[]ProviderAttr
		result, err := listAvailableProviders()
		if err != nil {
			return
		}
		for _, v := range *result {
			registryProviders = append(registryProviders, addrs.Provider{
				Type:        v.Name,
				Namespace:   v.Namespace,
				Hostname:    addrs.DefaultRegistryHost,
				Tier:        v.Tier,
				Description: v.Description,
			})
		}
	})

	return &registryProviders, err
}

func listAvailableProviders() (*[]ProviderAttr, error) {
	var result []ProviderAttr
	url := "/v2/providers?filter[tier]=official,partner,community&page[number]=1&page[size]=100"
	for url != "" {
		page, err := listProviders(url)
		if err != nil {
			return nil, err
		}
		for _, p := range page.Data {
			result = append(result, p.Attributes)
		}

		url = page.Links.Next
	}
	return &result, nil
}

func listProviders(uri string) (*ProviderPage, error) {
	if !strings.HasPrefix(uri, "https://") {
		uri = "https://registry.terraform.io" + uri
	}
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var page ProviderPage
	// UnknownFields are ignored
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	return &page, nil
}
