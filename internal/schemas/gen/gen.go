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

	log.Println("fetching verified partner providers from registry")
	partnerProviders, err := listProviders("partner")
	if err != nil {
		return err
	}

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

type registryResponse struct {
	Data []provider `json:"data"`
}

func listProviders(tier string) ([]provider, error) {
	// TODO will eventually need to paginate, for now "official" is 33 and "partner" is 95
	resp, err := http.Get(fmt.Sprintf("https://registry.terraform.io/v2/providers?page[size]=100&filter[tier]=%s", tier))
	if err != nil {
		return nil, err
	}

	var response registryResponse
	err = json.NewDecoder(resp.Body).Decode(&response)

	return filter(response.Data), err
}

var ignore = map[string]bool{
	"icinga2":  true,
	"vthunder": true,
	"sematext": true,
}

// certain providers are problematic/fail to download on terraform init
func filter(providers []provider) (filtered []provider) {
	for _, provider := range providers {
		if ok := ignore[provider.Attributes.Name]; ok {
			continue
		}
		filtered = append(filtered, provider)
	}
	return filtered
}
