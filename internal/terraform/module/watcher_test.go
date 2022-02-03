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
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/scheduler"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/stretchr/testify/mock"
)

func TestWatcher_initFromScratch(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(testLogger())

	fs := filesystem.NewFilesystem(ss.DocumentStore)

	modPath := filepath.Join(t.TempDir(), "module")
	err = os.Mkdir(modPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	psMock := &tfjson.ProviderSchemas{
		FormatVersion: "0.1",
		Schemas: map[string]*tfjson.ProviderSchema{
			"registry.terraform.io/hashicorp/aws": {},
		},
	}
	tfCalls := &exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			modPath: {
				{
					Method: "ProviderSchemas",
					Arguments: []interface{}{
						mock.AnythingOfType("*context.valueCtx"),
					},
					ReturnArguments: []interface{}{
						psMock,
						nil,
					},
				},
				{
					Method: "Version",
					Arguments: []interface{}{
						mock.AnythingOfType("*context.valueCtx"),
					},
					ReturnArguments: []interface{}{
						version.Must(version.NewVersion("1.0.0")),
						nil,
						nil,
					},
				},
			},
		},
	}

	ctx := context.Background()

	ctx = exec.WithExecutorOpts(ctx, &exec.ExecutorOpts{
		ExecPath: "tf-mock",
	})

	w, err := NewWatcher(fs, ss.Modules, ss.ProviderSchemas, ss.JobStore, exec.NewMockExecutor(tfCalls))
	if err != nil {
		t.Fatal(err)
	}
	w.SetLogger(testLogger())

	err = ss.Modules.Add(modPath)
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

	err = w.Start(ctx)
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

	jobIds, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("queued jobs: %q", jobIds)

	scheduler := scheduler.NewScheduler(&closedJobStore{ss.JobStore}, 1)
	scheduler.Start(ctx)
	t.Cleanup(scheduler.Stop)

	err = ss.JobStore.WaitForJobs(ctx, jobIds...)
	if err != nil {
		t.Fatal(err)
	}

	vc, err := version.NewConstraint("~> 3.0")
	if err != nil {
		t.Fatal(err)
	}
	ps, err := ss.ProviderSchemas.ProviderSchema(modPath, tfaddr.NewDefaultProvider("aws"), vc)
	if err != nil {
		t.Fatal(err)
	}
	expectedSchema := &tfschema.ProviderSchema{
		Resources:   map[string]*schema.BodySchema{},
		DataSources: map[string]*schema.BodySchema{},
	}
	if diff := cmp.Diff(expectedSchema, ps); diff != "" {
		t.Fatalf("schema mismatch: %s", diff)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if mod.TerraformVersion == nil {
		t.Fatal("expected non-nil version")
	}
	if mod.TerraformVersion.String() != "1.0.0" {
		t.Fatalf("version mismatch.\ngiven:   %q\nexpected: %q",
			mod.TerraformVersion.String(), "1.0.0")
	}
}

type closedJobStore struct {
	js *state.JobStore
}

func (js *closedJobStore) EnqueueJob(newJob job.Job) (job.ID, error) {
	return js.js.EnqueueJob(newJob)
}

func (js *closedJobStore) AwaitNextJob(ctx context.Context) (job.ID, job.Job, error) {
	return js.js.AwaitNextJob(ctx, false)
}

func (js *closedJobStore) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	return js.js.FinishJob(id, jobErr, deferredJobIds...)
}

func (js *closedJobStore) WaitForJobs(ctx context.Context, jobIds ...job.ID) error {
	return js.js.WaitForJobs(ctx, jobIds...)
}
