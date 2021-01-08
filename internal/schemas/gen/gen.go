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
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/shurcooL/vfsgen"
)

const terraformBlock = `terraform {
	required_version = "~> 0.13"
  required_providers {
  {{ range $p := . }}
    {{ $p.Name }} = {
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
	log.Printf("len=%d", len(providers))

	log.Println("fetching verified partner providers from registry")
	partnerProviders, err := listProviders("partner")
	if err != nil {
		return err
	}
	log.Printf("len=%d", len(partnerProviders))

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

	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	execPath, err := tfinstall.Find(ctx, tfinstall.LookPath(), tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		return err
	}

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

// these providers fail to download
// Error: Failed to query available provider packages

// Could not retrieve the list of available versions for provider icinga/icinga2:
// no available releases match the given constraints

// Error: Failed to install provider

// Error while installing a10networks/vthunder v0.4.21: could not query provider
// registry for registry.terraform.io/a10networks/vthunder: failed to retrieve
// authentication checksums for provider: 404 Not Found

// Error: Failed to install provider

// Error while installing sematext/sematext v0.1.9: checksum list has unexpected
// SHA-256 hash f323df96ca63ead7cd57e2f58e2061199cd36568837863fde44af2e60949c5c2
// (expected ce87fc7c44222b5f679d7dc1e2cbff7984e5a05fb011739ff92b747b39ea1528)
var ignore = map[string]bool{
	"icinga2":  true,
	"vthunder": true,
	"sematext": true,
}

func filter(providers []provider) (filtered []provider) {
	for _, provider := range providers {
		if ok := ignore[provider.Attributes.Name]; ok {
			continue
		}
		filtered = append(filtered, provider)
	}
	return filtered
}
