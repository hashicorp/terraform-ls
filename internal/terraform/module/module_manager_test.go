package module

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/scheduler"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestWalker_complexModules(t *testing.T) {
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		root             string
		totalModuleCount int

		expectedModules     []string
		expectedSchemaPaths []string
	}{
		{
			filepath.Join(testData, "single-root-ext-modules-only"),
			1,
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc1", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-ext-modules-only", ".terraform", "modules", "vpc2", "terraform-google-network-2.3.0", "modules", "vpc"),
			},
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
			},
		},

		{
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			1,
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "five", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "routes"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "subnets"),
				filepath.Join(testData, "single-root-local-and-ext-modules", ".terraform", "modules", "four", "terraform-google-network-2.3.0", "modules", "vpc"),
				filepath.Join(testData, "single-root-local-and-ext-modules", "alpha"),
				filepath.Join(testData, "single-root-local-and-ext-modules", "beta"),
			},
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},

		{
			filepath.Join(testData, "single-root-local-modules-only"),
			1,
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
				filepath.Join(testData, "single-root-local-modules-only", "alpha"),
				filepath.Join(testData, "single-root-local-modules-only", "beta"),
			},
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},

		{
			filepath.Join(testData, "single-root-no-modules"),
			1,
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-ext-modules-only"),
			1,
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			1,
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "alpha"),
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "charlie"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},

		{
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			1,
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module"),
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},

		// Multi-root

		{
			filepath.Join(testData, "main-module-multienv"),
			3,
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
				filepath.Join(testData, "main-module-multienv", "env", "staging"),
				filepath.Join(testData, "main-module-multienv", "main"),
				filepath.Join(testData, "main-module-multienv", "modules", "application"),
				filepath.Join(testData, "main-module-multienv", "modules", "database"),
			},
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
				filepath.Join(testData, "main-module-multienv", "env", "staging"),
			},
		},

		{
			filepath.Join(testData, "multi-root-no-modules"),
			3,
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
				filepath.Join(testData, "multi-root-no-modules", "third-root"),
			},
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
				filepath.Join(testData, "multi-root-no-modules", "third-root"),
			},
		},

		{
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "first-root", "alpha"),
				filepath.Join(testData, "multi-root-local-modules-down", "first-root", "charlie"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root", "alpha"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root", "charlie"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root", "alpha"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root", "charlie"),
			},
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
				filepath.Join(testData, "multi-root-local-modules-down", "third-root"),
			},
		},

		{
			filepath.Join(testData, "multi-root-local-modules-up"),
			3,
			[]string{
				filepath.Join(testData, "multi-root-local-modules-up", "main-module"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "first"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "second"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "third"),
			},
			[]string{
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "first"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "second"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "third"),
			},
		},
	}

	ctx := context.Background()

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.root), func(t *testing.T) {
			ss, err := state.NewStateStore()
			if err != nil {
				t.Fatal(err)
			}
			ss.SetLogger(testLogger())

			fs := filesystem.NewFilesystem(ss.DocumentStore)
			tfCalls := &exec.TerraformMockCalls{
				AnyWorkDir: validTfMockCalls(tc.totalModuleCount),
			}
			ctx = exec.WithExecutorOpts(ctx, &exec.ExecutorOpts{
				ExecPath: "tf-mock",
			})

			s := scheduler.NewScheduler(&closedJobStore{ss.JobStore}, 1)
			ss.SetLogger(testLogger())
			s.Start(ctx)

			pa := state.NewPathAwaiter(ss.WalkerPaths, false)
			w := NewWalker(fs, pa, ss.Modules, ss.ProviderSchemas, ss.JobStore, exec.NewMockExecutor(tfCalls))
			w.Collector = NewWalkerCollector()
			w.SetLogger(testLogger())
			dir := document.DirHandleFromPath(tc.root)
			err = ss.WalkerPaths.EnqueueDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			err = w.StartWalking(ctx)
			if err != nil {
				t.Fatal(err)
			}
			err = ss.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{dir})
			if err != nil {
				t.Fatal(err)
			}
			err = ss.JobStore.WaitForJobs(ctx, w.Collector.JobIds()...)
			if err != nil {
				t.Fatal(err)
			}
			err = w.Collector.ErrorOrNil()
			if err != nil {
				t.Fatal(err)
			}

			s.Stop()

			modules, err := ss.Modules.List()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedModules, modulePaths(modules)); diff != "" {
				t.Fatalf("modules don't match: %s", diff)
			}

			it, err := ss.ProviderSchemas.ListSchemas()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedSchemaPaths, localProviderSchemaPaths(t, it)); diff != "" {
				t.Fatalf("schemas don't match: %s", diff)
			}
		})
	}
}

func modulePaths(modules []*state.Module) []string {
	paths := make([]string, len(modules))

	for i, mod := range modules {
		paths[i] = mod.Path
	}

	return paths
}

func localProviderSchemaPaths(t *testing.T, it *state.ProviderSchemaIterator) []string {
	schemas := make([]string, 0)

	for ps := it.Next(); ps != nil; ps = it.Next() {
		localSrc, ok := ps.Source.(state.LocalSchemaSource)
		if !ok {
			t.Fatalf("expected only local sources, found: %q", ps.Source)
		}

		schemas = append(schemas, localSrc.ModulePath)
	}

	return schemas
}

func validTfMockCalls(repeatability int) []*mock.Call {
	return []*mock.Call{
		{
			Method:        "Version",
			Repeatability: repeatability,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.valueCtx"),
			},
			ReturnArguments: []interface{}{
				version.Must(version.NewVersion("0.12.0")),
				nil,
				nil,
			},
		},
		{
			Method:        "GetExecPath",
			Repeatability: repeatability,
			ReturnArguments: []interface{}{
				"",
			},
		},
		{
			Method:        "ProviderSchemas",
			Repeatability: repeatability,
			Arguments: []interface{}{
				mock.AnythingOfType("*context.valueCtx"),
			},
			ReturnArguments: []interface{}{
				testProviderSchema,
				nil,
			},
		},
	}
}

var testProviderSchema = &tfjson.ProviderSchemas{
	FormatVersion: "0.1",
	Schemas: map[string]*tfjson.ProviderSchema{
		"test": {
			ConfigSchema: &tfjson.Schema{},
		},
	},
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}

type closedJobStore struct {
	js *state.JobStore
}

func (js *closedJobStore) EnqueueJob(newJob job.Job) (job.ID, error) {
	return js.js.EnqueueJob(newJob)
}

func (js *closedJobStore) AwaitNextJob(ctx context.Context) (job.ID, job.Job, error) {
	return js.js.AwaitNextJob(ctx, job.LowPriority)
}

func (js *closedJobStore) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	return js.js.FinishJob(id, jobErr, deferredJobIds...)
}

func (js *closedJobStore) WaitForJobs(ctx context.Context, jobIds ...job.ID) error {
	return js.js.WaitForJobs(ctx, jobIds...)
}
