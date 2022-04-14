//go:build generate
// +build generate

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	hcinstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/shurcooL/vfsgen"
)

const terraformBlock = `terraform {
	required_version = "~> 1"
  required_providers {
  {{ range $i, $p := . }}
    {{ $p.Name }}-{{ $i }} = {
      source = "{{ $p.Source }}"
    }
  {{ end }}
  }
}
`

func main() {
	os.Exit(func() int {
		if err := gen(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}())
}

func gen() error {
	ctx := context.Background()

	log.Println("fetching official providers from registry")
	providers, err := listProviders("official")
	if err != nil {
		return err
	}
	log.Printf("fetched official providers: %d", len(providers))

	log.Println("fetching verified partner providers from registry")
	partnerProviders, err := listProviders("partner")
	if err != nil {
		return err
	}
	log.Printf("fetched partner providers: %d", len(partnerProviders))

	providers = append(providers, partnerProviders...)

	log.Println("parsing template")
	tmpl, err := template.New("providers").Parse(terraformBlock)
	if err != nil {
		return err
	}

	log.Println("creating config file")
	configFile, err := os.Create("providers.tf")
	if err != nil {
		return err
	}

	log.Println("executing template")
	err = tmpl.Execute(configFile, providers)
	if err != nil {
		return err
	}

	log.Println("ensuring terraform is installed")

	installDir, err := ioutil.TempDir("", "hcinstall")
	if err != nil {
		return err
	}
	defer os.RemoveAll(installDir)

	i := hcinstall.NewInstaller()
	execPath, err := i.Ensure(ctx, []src.Source{
		&fs.AnyVersion{
			Product: &product.Terraform,
		},
		&releases.LatestVersion{
			Product:    product.Terraform,
			InstallDir: installDir,
		},
	})
	if err != nil {
		return err
	}

	defer i.Remove(ctx)

	log.Println("running terraform init")

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	tf, err := tfexec.NewTerraform(cwd, execPath)
	if err != nil {
		return err
	}

	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return err
	}

	log.Println("creating schemas/data dir")
	err = os.MkdirAll("data", 0755)
	if err != nil {
		return err
	}
	fs := http.Dir("data")

	coreVersion, providerVersions, err := tf.Version(ctx, true)
	if err != nil {
		return err
	}

	log.Println("creating version file")
	versionFile, err := os.Create(filepath.Join("data", "versions.json"))
	if err != nil {
		return err
	}
	versionOutput := &schemas.RawVersionOutput{
		CoreVersion: coreVersion.String(),
		Providers:   stringifyProviderVersions(providerVersions),
	}

	log.Println("writing versions to file")
	err = json.NewEncoder(versionFile).Encode(versionOutput)
	if err != nil {
		return err
	}

	// TODO upstream change to have tfexec write to file directly instead of unmarshal/remarshal
	log.Println("running terraform providers schema")
	ps, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return err
	}

	log.Println("creating schemas file")
	schemasFile, err := os.Create(filepath.Join("data", "schemas.json"))
	if err != nil {
		return err
	}

	log.Println("writing schemas to file")
	err = json.NewEncoder(schemasFile).Encode(ps)
	if err != nil {
		return err
	}

	log.Println("generating embedded go file")
	return vfsgen.Generate(fs, vfsgen.Options{
		Filename:     "schemas_gen.go",
		PackageName:  "schemas",
		VariableName: "files",
	})
}

func stringifyProviderVersions(m map[string]*version.Version) map[string]string {
	versions := make(map[string]string, 0)

	for addr, ver := range m {
		versions[addr] = ver.String()
	}

	return versions
}

type providerAttributes struct {
	Name     string `json:"name"`
	FullName string `json:"full-name"`
}

type provider struct {
	Attributes providerAttributes `json:"attributes"`
}

func (p provider) Name() string {
	return p.Attributes.Name
}

func (p provider) Source() string {
	// terraform provider is builtin and has special source
	if p.Attributes.Name == "terraform" {
		return "terraform.io/builtin/terraform"
	}
	return p.Attributes.FullName
}

type pagination struct {
	NextPage int `json:"next-page"`
}

type meta struct {
	Pagination pagination `json:"pagination"`
}

type registryResponse struct {
	Data []provider `json:"data"`
	Meta meta       `json:"meta"`
}

func listProviders(tier string) ([]provider, error) {
	var providers []provider
	page := 1
	for page > 0 {
		resp, err := http.Get(fmt.Sprintf("https://registry.terraform.io/v2/providers?page[size]=100&filter[tier]=%s&page[number]=%d", tier, page))
		if err != nil {
			return nil, err
		}

		var response registryResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, err
		}
		providers = append(providers, response.Data...)
		page = response.Meta.Pagination.NextPage
	}
	return filter(providers), nil
}

var ignore = map[string]bool{
	"a10networks/vthunder":         true,
	"delphix-integrations/delphix": true,
	"harness-io/harness":           true,
	"harness/harness-platform":     true,
	"HewlettPackard/oneview":       true,
	"HewlettPackard/hpegl":         true,
	"jradtilbrook/buildkite":       true,
	"kvrhdn/honeycombio":           true,
	"ThalesGroup/ciphertrust":      true,
	"nullstone-io/ns":              true,
	"zededa/zedcloud":              true,
}

func filter(providers []provider) (filtered []provider) {
	for _, provider := range providers {
		if ok := ignore[provider.Source()]; ok {
			continue
		}
		filtered = append(filtered, provider)
	}
	return filtered
}
