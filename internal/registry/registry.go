package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

const (
	defaultBaseURL = "https://registry.terraform.io"
	defaultTimeout = 5 * time.Second
)

type Client struct {
	BaseURL string
	Timeout time.Duration
}

func NewClient() Client {
	return Client{
		BaseURL: defaultBaseURL,
		Timeout: defaultTimeout,
	}
}

func (c Client) GetModuleData(addr tfaddr.Module, cons version.Constraints) (*ModuleResponse, error) {
	var response ModuleResponse

	v, err := c.GetMatchingModuleVersion(addr, cons)
	if err != nil {
		return nil, err
	}

	client := cleanhttp.DefaultClient()
	client.Timeout = defaultTimeout

	url := fmt.Sprintf("%s/v1/modules/%s/%s/%s/%s", c.BaseURL,
		addr.Package.Namespace,
		addr.Package.Name,
		addr.Package.TargetSystem,
		v.String())
	resp, err := client.Get(url)
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

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c Client) GetMatchingModuleVersion(addr tfaddr.Module, con version.Constraints) (*version.Version, error) {
	url := fmt.Sprintf("%s/v1/modules/%s/%s/%s/versions", c.BaseURL,
		addr.Package.Namespace,
		addr.Package.Name,
		addr.Package.TargetSystem)

	client := cleanhttp.DefaultClient()
	client.Timeout = defaultTimeout

	resp, err := client.Get(url)
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

	var response ModuleVersionsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	var foundVersions version.Collection
	for _, module := range response.Modules {
		for _, entry := range module.Versions {
			ver, err := version.NewVersion(entry.Version)
			if err == nil {
				foundVersions = append(foundVersions, ver)
			}
		}
	}

	sort.Sort(foundVersions)

	for _, fv := range foundVersions {
		if con.Check(fv) {
			return fv, nil
		}
	}

	return nil, fmt.Errorf("no suitable version found for %q %q", addr, con)
}
