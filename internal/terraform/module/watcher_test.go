package module

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestWatcher_initFromScratch(t *testing.T) {
	fs := filesystem.NewFilesystem()

	modPath := filepath.Join(t.TempDir(), "module")
	err := os.Mkdir(modPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	psMock := &tfjson.ProviderSchemas{
		FormatVersion: "0.1",
		Schemas: map[string]*tfjson.ProviderSchema{
			"custom": {},
		},
	}
	mmm := NewModuleManagerMock(&ModuleManagerMockInput{
		Logger: testLogger(),
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				modPath: {
					{
						Method: "ProviderSchemas",
						Arguments: []interface{}{
							mock.AnythingOfType("*context.cancelCtx"),
						},
						ReturnArguments: []interface{}{
							psMock,
							nil,
						},
					},
					{
						Method: "Version",
						Arguments: []interface{}{
							mock.AnythingOfType("*context.cancelCtx"),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("1.0.0")),
							nil,
							nil,
						},
					},
				},
			},
		},
	})
	ctx := context.Background()
	modMgr := mmm(ctx, fs)

	w, err := NewWatcher(fs, modMgr)
	if err != nil {
		t.Fatal(err)
	}
	w.SetLogger(testLogger())

	mod, err := modMgr.AddModule(modPath)
	if err != nil {
		t.Fatal(err)
	}

	b := []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_vpc" "example" {
  cidr_block = "10.0.0.0/16"
}
`)
	err = ioutil.WriteFile(filepath.Join(modPath, "main.tf"), b, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = w.AddModule(modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = w.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		w.Stop()
	})

	err = ioutil.WriteFile(filepath.Join(modPath, ".terraform.lock.hcl"), b, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Give watcher some time to react
	time.Sleep(250 * time.Millisecond)

	ps, err := mod.ProviderSchema()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(psMock, ps); diff != "" {
		t.Fatalf("schema mismatch: %s", diff)
	}

	v, err := mod.TerraformVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v == nil {
		t.Fatal("expected non-nil version")
	}
	if v.String() != "1.0.0" {
		t.Fatalf("version mismatch.\ngiven:   %q\nexpected: %q",
			v.String(), "1.0.0")
	}
}
