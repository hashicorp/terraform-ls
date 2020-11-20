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

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/shurcooL/vfsgen"
)

const terraformBlock = `terraform {
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

	err = tf.Init(ctx, tfexec.Upgrade(true), tfexec.LockTimeout("120s"))
	if err != nil {
		return err
	}

	// TODO upstream change to have tfexec write to file directly instead of unmarshal/remarshal
	log.Println("running terraform providers schema")
	ps, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return err
	}

	log.Println("creating schemas/data dir")
	err = os.MkdirAll("data", 0755)
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
	var fs http.FileSystem = http.Dir("data")
	return vfsgen.Generate(fs, vfsgen.Options{
		Filename:     "schemas.go",
		PackageName:  "schemas",
		VariableName: "Files",
	})
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
	// TODO will eventually need to paginate, for now "official" is 33 and "partner" is 82
	resp, err := http.Get(fmt.Sprintf("https://registry.terraform.io/v2/providers?page[size]=100&filter[tier]=%s", tier))
	if err != nil {
		return nil, err
	}

	var response registryResponse
	err = json.NewDecoder(resp.Body).Decode(&response)

	return response.Data, err
}
