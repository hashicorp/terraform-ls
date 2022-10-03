//go:build generate
// +build generate

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/hashicorp/go-version"
	hcinstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/registry"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

var terraformVersion = version.MustConstraints(version.NewConstraint("~> 1.0"))

type Provider struct {
	ID   string
	Addr tfaddr.Provider
}

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

	ctx, cancelFunc := lsctx.WithSignalCancel(context.Background(), log.Default(),
		os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	providers := make([]Provider, 0)
	providers = append(providers, Provider{
		ID:   "0",
		Addr: tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform"),
	})

	// obtain all official & partner providers from the Registry
	client := registry.NewClient()
	log.Println("fetching official providers from registry")
	officialProviders, err := client.ListProviders("official")
	if err != nil {
		return err
	}
	log.Printf("fetched official providers: %d", len(officialProviders))
	for _, p := range officialProviders {
		if p.Attributes.Namespace == "hashicorp" && p.Attributes.Name == "terraform" {
			// skip the old terraform provider as this is now built-in
			continue
		}
		providers = append(providers, Provider{
			ID: p.ID,
			Addr: tfaddr.NewProvider(
				tfaddr.DefaultProviderRegistryHost,
				p.Attributes.Namespace,
				p.Attributes.Name,
			),
		})
	}
	log.Println("fetching verified partner providers from registry")
	partnerProviders, err := client.ListProviders("partner")
	if err != nil {
		return err
	}
	log.Printf("fetched partner providers: %d", len(partnerProviders))
	for _, p := range partnerProviders {
		providers = append(providers, Provider{
			ID: p.ID,
			Addr: tfaddr.NewProvider(
				tfaddr.DefaultProviderRegistryHost,
				p.Attributes.Namespace,
				p.Attributes.Name,
			),
		})
	}

	// find or install Terraform
	log.Println("ensuring terraform is installed")
	installDir, err := ioutil.TempDir("", "hcinstall")
	if err != nil {
		return err
	}
	defer os.RemoveAll(installDir)
	i := hcinstall.NewInstaller()
	execPath, err := i.Ensure(ctx, []src.Source{
		&releases.LatestVersion{
			Product:     product.Terraform,
			InstallDir:  installDir,
			Constraints: terraformVersion,
		},
	})
	if err != nil {
		return err
	}
	defer i.Remove(ctx)

	// log version
	tf, err := tfexec.NewTerraform(installDir, execPath)
	if err != nil {
		return err
	}
	coreVersion, _, err := tf.Version(ctx, true)
	if err != nil {
		return err
	}
	log.Printf("installed Terraform %s", coreVersion)

	workspacePath, err := filepath.Abs("gen-workspace")
	if err != nil {
		return err
	}
	dataDirPath, err := filepath.Abs("data")
	if err != nil {
		return err
	}

	// remove data from previous run
	err = os.RemoveAll(workspacePath)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dataDirPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			// ensure that git.keep is kept in place
			continue
		}
		err = os.RemoveAll(filepath.Join(dataDirPath, entry.Name()))
		if err != nil {
			return err
		}
	}

	// install each provider and obtain schema for it
	var wg sync.WaitGroup
	for _, p := range providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()

			log.Printf("%s: obtaining schema ...", p.Addr.ForDisplay())
			details, err := schemaForProvider(ctx, client, Inputs{
				TerraformExecPath: execPath,
				WorkspacePath:     workspacePath,
				DataDirPath:       dataDirPath,
				CoreVersion:       coreVersion,
				Provider:          p,
			})
			if err != nil {
				log.Printf("%s: %s", p.Addr.ForDisplay(), err)
				return
			}

			log.Printf("%s: obtained schema for %s (%d bytes); terraform init: %s",
				p.Addr.ForDisplay(), details.Version,
				details.Size, details.InitElapsedTime)
		}(p)
	}
	wg.Wait()

	return nil
}

type Inputs struct {
	TerraformExecPath string
	WorkspacePath     string
	DataDirPath       string
	CoreVersion       *version.Version
	Provider          Provider
}

type Outputs struct {
	Version         string
	Size            int64
	InitElapsedTime time.Duration
}

func schemaForProvider(ctx context.Context, client registry.Client, input Inputs) (*Outputs, error) {
	var pVersion *version.Version
	if input.Provider.Addr.IsBuiltIn() {
		pVersion = input.CoreVersion
	} else {
		resp, err := client.GetLatestProviderVersion(input.Provider.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest version: %w", err)
		}

		pVersion, err = version.NewVersion(resp.Data.Attributes.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q: %w", resp.Data.Attributes.Version, err)
		}

		if !providerVersionSupportsOsAndArch(resp.Included, runtime.GOOS, runtime.GOARCH) {
			return nil, fmt.Errorf("version %s does not support %s/%s", pVersion, runtime.GOOS, runtime.GOARCH)
		}
	}

	wd := filepath.Join(input.WorkspacePath,
		input.Provider.Addr.Hostname.String(),
		input.Provider.Addr.Namespace,
		input.Provider.Addr.Type,
		pVersion.String())
	err := os.MkdirAll(wd, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create workspace dir: %w", err)
	}

	dataDir := filepath.Join(input.DataDirPath,
		input.Provider.Addr.Hostname.String(),
		input.Provider.Addr.Namespace,
		input.Provider.Addr.Type,
		pVersion.String())
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create data dir: %w", err)
	}

	type templateData struct {
		TerraformVersion string
		LocalName        string
		Source           string
		Version          string
	}
	tmpl, err := template.New("providers").Parse(`terraform {
  required_version = "{{ .TerraformVersion }}"
  required_providers {
    {{ .LocalName }} = {
      source  = "{{ .Source }}"
      {{ with .Version }}version = "{{ . }}"{{ end }}
    }
  }
}
`)
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %w", err)
	}

	versionFilePath := filepath.Join(wd, "versions.tf")
	configFile, err := os.Create(versionFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create config file: %w", err)
	}

	err = tmpl.Execute(configFile, templateData{
		TerraformVersion: terraformVersion.String(),
		LocalName:        "provider" + input.Provider.ID,
		Source:           input.Provider.Addr.ForDisplay(),
		Version:          pVersion.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	configFile.Close()

	tf, err := tfexec.NewTerraform(wd, input.TerraformExecPath)
	if err != nil {
		return nil, err
	}

	var initElapsed time.Duration
	if !input.Provider.Addr.IsBuiltIn() {
		initElapsed, err = retryInit(ctx, tf, input.Provider.Addr.ForDisplay(), 0)
		if err != nil {
			return nil, err
		}

		_, pVersions, err := tf.Version(ctx, true)
		if err != nil {
			return nil, err
		}

		pv, ok := pVersions[input.Provider.Addr.String()]
		if !ok {
			return nil, fmt.Errorf("provider version not found for %q", input.Provider.Addr.ForDisplay())
		}
		if !pv.Equal(pVersion) {
			return nil, fmt.Errorf("expected provider version %s to match %s", pv, pVersion)
		}
	}

	// TODO upstream change to have tfexec write to file directly instead of unmarshal/remarshal
	ps, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(filepath.Join(dataDir, "schema.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create schema file: %w", err)
	}

	err = json.NewEncoder(f).Encode(ps)
	if err != nil {
		return nil, fmt.Errorf("failed to encode schema file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to check schema file: %w", err)
	}

	return &Outputs{
		Version:         pVersion.String(),
		Size:            fi.Size(),
		InitElapsedTime: initElapsed,
	}, nil
}

// retryInit runs "terraform init" and attempts to retry
// on known (typically network-related) transient errors
func retryInit(ctx context.Context, tf *tfexec.Terraform, fullName string, retried int) (time.Duration, error) {
	maxRetries := 5
	backoffPeriod := 2 * time.Second

	startTime := time.Now()
	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		if retried >= maxRetries {
			timeElapsed := time.Now().Sub(startTime)
			return timeElapsed, fmt.Errorf("%s: final error after 5 retries: %w", fullName, err)
		}

		if shortErr, ok := initErrorIsRetryable(err); ok {
			log.Printf("%s: %s", fullName, err)
			retried++
			log.Printf("%s: will retry init (attempt %d) in %s due to %s", fullName, retried, backoffPeriod, shortErr)
			time.Sleep(backoffPeriod)
			return retryInit(ctx, tf, fullName, retried)
		}
		return 0, err
	}

	timeElapsed := time.Now().Sub(startTime)
	return timeElapsed, nil
}

func initErrorIsRetryable(err error) (string, bool) {
	if strings.Contains(err.Error(), "i/o timeout") {
		return "i/o timeout", true
	}
	if strings.Contains(err.Error(), "request canceled while waiting for connection") {
		return "connection timeout", true
	}
	if strings.Contains(err.Error(), "handshake timeout") {
		return "handshake timeout", true
	}
	if strings.Contains(err.Error(), "no route to host") {
		return "no route to host", true
	}
	if strings.Contains(err.Error(), "context deadline exceeded") {
		return "context deadline exceeded", true
	}
	if strings.Contains(err.Error(), "503 Service Unavailable") {
		return "503 Service Unavailable", true
	}
	return "", false
}

func providerVersionSupportsOsAndArch(includes []registry.Included, os, arch string) bool {
	for _, inc := range includes {
		if inc.Type == "provider-platforms" &&
			inc.Attributes.Os == os &&
			inc.Attributes.Arch == arch {
			return true
		}
	}
	return false
}
