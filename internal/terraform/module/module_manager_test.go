package module

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TestModuleManager_ModuleCandidatesByPath(t *testing.T) {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// The underlying API is now deprecated anyway
		// so it's not worth adapting tests for all platforms.
		// We just skip tests on Apple Silicon.
		t.Skip("deprecated API")
	}
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name               string
		walkerRoot         string
		totalModuleCount   int
		lookupPath         string
		expectedCandidates []string
	}{
		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-ext-modules-only"),
			1,
			filepath.Join(testData, "single-root-ext-modules-only"),
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
			},
		},

		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			1,
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			1,
			filepath.Join(testData, "single-root-local-and-ext-modules/alpha"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			1,
			filepath.Join(testData, "single-root-local-and-ext-modules/beta"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"mod-ref-based lookup (not referenced)",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			1,
			filepath.Join(testData, "single-root-local-and-ext-modules/charlie"),
			[]string{},
		},

		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-local-modules-only"),
			1,
			filepath.Join(testData, "single-root-local-modules-only"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-modules-only"),
			1,
			filepath.Join(testData, "single-root-local-modules-only/alpha"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-modules-only"),
			1,
			filepath.Join(testData, "single-root-local-modules-only/beta"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"mod-ref-based lookup (not referenced)",
			filepath.Join(testData, "single-root-local-modules-only"),
			1,
			filepath.Join(testData, "single-root-local-modules-only/charlie"),
			[]string{},
		},

		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-no-modules"),
			1,
			filepath.Join(testData, "single-root-no-modules"),
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
		},

		{
			"directory-based lookup",
			filepath.Join(testData, "nested-single-root-no-modules"),
			1,
			filepath.Join(testData, "nested-single-root-no-modules", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-no-modules", "tf-root"),
			},
		},

		{
			"directory-based lookup",
			filepath.Join(testData, "nested-single-root-ext-modules-only"),
			1,
			filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
		},

		{
			"directory-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			1,
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			1,
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "alpha"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			1,
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "beta"),
			[]string{},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			1,
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "charlie"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			1,
			filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			1,
			filepath.Join(testData, "nested-single-root-local-modules-up", "module"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},

		// Multi-root

		{
			"directory-env-based lookup",
			filepath.Join(testData, "main-module-multienv"),
			3,
			filepath.Join(testData, "main-module-multienv", "env", "dev"),
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
			},
		},
		{
			"directory-env-based lookup",
			filepath.Join(testData, "main-module-multienv"),
			3,
			filepath.Join(testData, "main-module-multienv", "env", "prod"),
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
			},
		},
		{
			"main module lookup",
			filepath.Join(testData, "main-module-multienv"),
			3,
			filepath.Join(testData, "main-module-multienv", "main"),
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
				filepath.Join(testData, "main-module-multienv", "env", "staging"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-no-modules"),
			3,
			filepath.Join(testData, "multi-root-no-modules", "first-root"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
			},
		},
		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-no-modules"),
			3,
			filepath.Join(testData, "multi-root-no-modules", "second-root"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "first-root", "alpha"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "first-root", "beta"),
			[]string{},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "first-root", "charlie"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "second-root", "alpha"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "second-root", "beta"),
			[]string{},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			3,
			filepath.Join(testData, "multi-root-local-modules-down", "second-root", "charlie"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-local-modules-up"),
			3,
			filepath.Join(testData, "multi-root-local-modules-up", "main-module"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "first"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "second"),
				filepath.Join(testData, "multi-root-local-modules-up", "main-module", "modules", "third"),
			},
		},
	}

	for i, tc := range testCases {
		base := filepath.Base(tc.walkerRoot)
		t.Run(fmt.Sprintf("%d-%s/%s", i, tc.name, base), func(t *testing.T) {
			ctx := context.Background()
			fs := filesystem.NewFilesystem()
			mmock := NewModuleManagerMock(&ModuleManagerMockInput{
				Logger: testLogger(),
				TerraformCalls: &exec.TerraformMockCalls{
					AnyWorkDir: validTfMockCalls(tc.totalModuleCount),
				},
			})
			ss, err := state.NewStateStore()
			if err != nil {
				t.Fatal(err)
			}
			mm := mmock(ctx, fs, ss.Modules, ss.ProviderSchemas)
			t.Cleanup(mm.CancelLoading)

			w := SyncWalker(fs, mm, noopServer{})
			w.SetLogger(testLogger())
			w.EnqueuePath(tc.walkerRoot)
			err = w.StartWalking(ctx)
			if err != nil {
				t.Fatal(err)
			}

			mm.AddModule(tc.lookupPath)

			candidates, err := mm.SchemaSourcesForModule(tc.lookupPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedCandidates, schemaSourcesPaths(t, candidates)); diff != "" {
				t.Fatalf("candidates don't match: %s", diff)
			}
		})
	}
}

type noopServer struct{}

func (s noopServer) Notify(ctx context.Context, method string, params interface{}) error {
	return nil
}

func (s noopServer) Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error) {
	return nil, nil
}

func schemaSourcesPaths(t *testing.T, srcs []SchemaSource) []string {
	paths := make([]string, len(srcs))
	for i, src := range srcs {
		paths[i] = src.Path
	}

	return paths
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}
