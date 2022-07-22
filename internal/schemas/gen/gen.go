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
	"runtime"
	"strings"

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

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	tf, err := tfexec.NewTerraform(cwd, execPath)
	if err != nil {
		return err
	}

	coreVersion, _, err := tf.Version(ctx, false)
	if err != nil {
		return err
	}
	log.Printf("using Terraform %s", coreVersion)

	log.Println("running terraform init")

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
	"hewlettpackard/oneview":       true,
	"hewlettpackard/hpegl":         true,
	"jradtilbrook/buildkite":       true,
	"kvrhdn/honeycombio":           true,
	"logicmonitor/logicmonitor":    true,
	"thalesgroup/ciphertrust":      true,
	"nullstone-io/ns":              true,
	"zededa/zedcloud":              true,
	"lightstep/lightstep":          true,
	"thousandeyes/thousandeyes":    true,
}

var darwinArm64Ignore = map[string]bool{
	"a10networks/thunder":         true,
	"alertmixer/amixr":            true,
	"aristanetworks/cloudvision":  true,
	"bluecatlabs/bluecat":         true,
	"ciscodevnet/ciscoasa":        true,
	"ciscodevnet/mso":             true,
	"ciscodevnet/sdwan":           true,
	"cloudtamer-io/cloudtamerio":  true,
	"cohesity/cohesity":           true,
	"commvault/commvault":         true,
	"consensys/quorum":            true,
	"f5networks/bigip":            true,
	"gocachebr/gocache":           true,
	"hashicorp/opc":               true,
	"hashicorp/oraclepaas":        true,
	"hashicorp/template":          true,
	"icinga/icinga2":              true,
	"infobloxopen/infoblox":       true,
	"infracost/infracost":         true,
	"instaclustr/instaclustr":     true,
	"ionos-cloud/profitbricks":    true,
	"juniper/junos-vsrx":          true,
	"llnw/limelight":              true,
	"netapp/netapp-elementsw":     true,
	"nirmata/nirmata":             true,
	"nttcom/ecl":                  true,
	"nutanix/nutanixkps":          true,
	"oktadeveloper/oktaasa":       true,
	"phoenixnap/pnap":             true,
	"purestorage-openconnect/cbs": true,
	"rafaysystems/rafay":          true,
	"rundeck/rundeck":             true,
	"sematext/sematext":           true,
	"skytap/skytap":               true,
	"splunk/synthetics":           true,
	"splunk/victorops":            true,
	"statuscakedev/statuscake":    true,
	"transloadit/transloadit":     true,
	"valtix-security/valtix":      true,
	"vmware-tanzu/carvel":         true,
	"wallix/waapm":                true,
	"william20111/thousandeyes":   true,
}

func filter(providers []provider) (filtered []provider) {
	for _, provider := range providers {
		src := strings.ToLower(provider.Source())
		if ok := ignore[src]; ok {
			continue
		}
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			if ok := darwinArm64Ignore[src]; ok {
				continue
			}
		}
		filtered = append(filtered, provider)
	}
	return filtered
}
