package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
)

type TerraformRegistryModule struct {
	Version   string     `json:"version"`
	Root      ModuleRoot `json:"root"`
}

type ModuleRoot struct {
	Inputs               []Input       `json:"inputs"`
	Outputs              []Output      `json:"outputs"`
}

type TerraformRegistryModuleVersions struct {
	Modules []struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	} `json:"modules"`
}

type RegistryModuleVersion struct {
	Version string `json:"version"`
	Root    struct {
		Providers []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			Source    string `json:"source"`
			Version   string `json:"version"`
		} `json:"providers"`
		Dependencies []interface{} `json:"dependencies"`
	} `json:"root"`
	Submodules []interface{} `json:"submodules"`
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

func GetTFRegistryInfo(p tfaddr.ModuleSourceRegistry, c module.DeclaredModuleCall) (*TerraformRegistryModule, error) {
	var response TerraformRegistryModule

	// modify this to first call https://github.com/hashicorp/terraform-registry/blob/main/docs/api/v1/modules.md#list-module-versions
	// to find version that matches constraint up above
	// then pull info
	v, err := GetVersion(p, c.Version)
	if err != nil {
		return nil, err
	}

	// get info on specific module
	url := fmt.Sprintf("https://registry.terraform.io/v1/modules/%s/%s", p.ForDisplay(), v.String())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func GetVersion(p tfaddr.ModuleSourceRegistry, con version.Constraints) (*version.Version, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/modules/%s/%s/%s/versions",
		p.PackageAddr.Namespace, p.PackageAddr.Name, p.PackageAddr.TargetSystem,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var things TerraformRegistryModuleVersions
	err = json.NewDecoder(resp.Body).Decode(&things)
	if err != nil {
		return nil, err
	}

	var foundVersions version.Collection
	for _, v := range things.Modules {
		for _, t := range v.Versions {
			g, _ := version.NewVersion(t.Version)
			foundVersions = append(foundVersions, g)
		}
	}

	sort.Sort(foundVersions)

	for _, fv := range foundVersions {
		if con.Check(fv) {
			return fv, nil
		}
	}

	return nil, fmt.Errorf("no versions found for constraint: %v", con)
}
