package rootmodule

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

// func TestRootModuleManager_RootModuleByPath_basic(t *testing.T) {
// 	rmm := testRootModuleManager(t)

// 	tmpDir := tempDir(t)
// 	direct, unrelated, dirbased := newTestRootModule(t, "direct"), newTestRootModule(t, "unrelated"), newTestRootModule(t, "dirbased")
// 	rmm.rms = map[string]*rootModule{
// 		direct.Dir:    direct.RootModule,
// 		unrelated.Dir: unrelated.RootModule,
// 		dirbased.Dir:  dirbased.RootModule,
// 	}

// 	w1, err := rmm.RootModuleByPath(direct.Dir)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if direct.RootModule != w1 {
// 		t.Fatalf("unexpected root module found: %p, expected: %p", w1, direct)
// 	}

// 	lockDirPath := filepath.Join(tmpDir, "dirbased", ".terraform", "plugins")
// 	lockFilePath := filepath.Join(lockDirPath, "selections.json")
// 	err = os.MkdirAll(lockDirPath, 0775)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	f, err := os.Create(lockFilePath)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	f.Close()

// 	w2, err := rmm.RootModuleByPath(lockFilePath)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if dirbased.RootModule != w2 {
// 		t.Fatalf("unexpected root module found: %p, expected: %p", w2, dirbased)
// 	}
// }

func TestRootModuleManager_RootModuleCandidatesByPath(t *testing.T) {
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct{
		name string
		walkerRoot string
		lookupPath string
		expectedCandidates []string
	}{
		{
			// outside of watcher, root modules are always looked up by dir
			"tf-file-based lookup",
			filepath.Join(testData, "single-root-ext-modules-only"),
			filepath.Join(testData, "single-root-ext-modules-only", "main.tf"),
			[]string{},
		},
		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-ext-modules-only"),
			filepath.Join(testData, "single-root-ext-modules-only"),
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "single-root-ext-modules-only"),
			filepath.Join(testData, "single-root-ext-modules-only", 
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "single-root-ext-modules-only"),
			},
		},

		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			filepath.Join(testData, "single-root-local-and-ext-modules",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			filepath.Join(testData, "single-root-local-and-ext-modules/alpha"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			filepath.Join(testData, "single-root-local-and-ext-modules/beta"),
			[]string{
				filepath.Join(testData, "single-root-local-and-ext-modules"),
			},
		},
		{
			"mod-ref-based lookup (not referenced)",
			filepath.Join(testData, "single-root-local-and-ext-modules"),
			filepath.Join(testData, "single-root-local-and-ext-modules/charlie"),
			[]string{},
		},

		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-local-modules-only"),
			filepath.Join(testData, "single-root-local-modules-only"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "single-root-local-modules-only"),
			filepath.Join(testData, "single-root-local-modules-only",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-modules-only"),
			filepath.Join(testData, "single-root-local-modules-only/alpha"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"mod-ref-based lookup",
			filepath.Join(testData, "single-root-local-modules-only"),
			filepath.Join(testData, "single-root-local-modules-only/beta"),
			[]string{
				filepath.Join(testData, "single-root-local-modules-only"),
			},
		},
		{
			"mod-ref-based lookup (not referenced)",
			filepath.Join(testData, "single-root-local-modules-only"),
			filepath.Join(testData, "single-root-local-modules-only/charlie"),
			[]string{},
		},

		{
			"dir-based lookup (exact match)",
			filepath.Join(testData, "single-root-no-modules"),
			filepath.Join(testData, "single-root-no-modules"),
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "single-root-no-modules"),
			filepath.Join(testData, "single-root-no-modules",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "single-root-no-modules"),
			},
		},

		{
			"directory-based lookup",
			filepath.Join(testData, "nested-single-root-no-modules"),
			filepath.Join(testData, "nested-single-root-no-modules", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-no-modules", "tf-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "nested-single-root-no-modules"),
			filepath.Join(testData, "nested-single-root-no-modules", "tf-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "nested-single-root-no-modules", "tf-root"),
			},
		},

		{
			"directory-based lookup",
			filepath.Join(testData, "nested-single-root-ext-modules-only"),
			filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "nested-single-root-ext-modules-only"),
			filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "nested-single-root-ext-modules-only", "tf-root"),
			},
		},

		{
			"directory-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "alpha"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "beta"),
			[]string{},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-down"),
			filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root", "charlie"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-down", "tf-root"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "nested-single-root-local-modules-up"),
			filepath.Join(testData, "nested-single-root-local-modules-up", "module"),
			[]string{
				filepath.Join(testData, "nested-single-root-local-modules-up", "module", "tf-root"),
			},
		},

		// Multi-root

		{
			"directory-env-based lookup",
			filepath.Join(testData, "main-module-multienv"),
			filepath.Join(testData, "main-module-multienv", "env", "dev"),
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "dev"),
			},
		},
		{
			"directory-env-based lookup",
			filepath.Join(testData, "main-module-multienv"),
			filepath.Join(testData, "main-module-multienv", "env", "prod"),
			[]string{
				filepath.Join(testData, "main-module-multienv", "env", "prod"),
			},
		},
		{
			"main module lookup",
			filepath.Join(testData, "main-module-multienv"),
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
			filepath.Join(testData, "multi-root-no-modules", "first-root"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
			},
		},
		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-no-modules"),
			filepath.Join(testData, "multi-root-no-modules", "second-root"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "multi-root-no-modules"),
			filepath.Join(testData, "multi-root-no-modules", "first-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "first-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "multi-root-no-modules"),
			filepath.Join(testData, "multi-root-no-modules", "second-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "multi-root-no-modules", "second-root"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "first-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "first-root", "alpha"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "first-root", "beta"),
			[]string{},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "first-root", "charlie"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "first-root"),
			},
		},
		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},
		{
			"lock-file-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "second-root",
				".terraform",
				"modules",
				"modules.json"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "second-root", "alpha"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "second-root", "beta"),
			[]string{},
		},
		{
			"mod-based lookup",
			filepath.Join(testData, "multi-root-local-modules-down"),
			filepath.Join(testData, "multi-root-local-modules-down", "second-root", "charlie"),
			[]string{
				filepath.Join(testData, "multi-root-local-modules-down", "second-root"),
			},
		},

		{
			"dir-based lookup",
			filepath.Join(testData, "multi-root-local-modules-up"),
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
		t.Run(fmt.Sprintf("%s/%d-%s", base, i, tc.name), func(t *testing.T) {
			rmm := testRootModuleManager(t)
			w := NewWalker()
			err := w.WalkInitializedRootModules(tc.walkerRoot, func(rmPath string) error {
				return rmm.AddRootModule(rmPath)
			})
			if err != nil {
				t.Fatal(err)
			}

			candidates := rmm.RootModuleCandidatesByPath(tc.lookupPath)
			if diff := cmp.Diff(tc.expectedCandidates, candidates); diff != "" {
				t.Fatalf("candidates don't match: %s", diff)
			}
		})
	}
}

func testRootModuleManager(t *testing.T) *rootModuleManager {
	rmm := newRootModuleManager(context.Background())
	rmm.logger = testLogger()
	rmm.newRootModule = func(ctx context.Context, dir string) (*rootModule, error) {
		rm := NewRootModuleMock(ctx, &RootModuleMock{
			TerraformExecQueue: &exec.MockQueue{
				Q: []*exec.MockItem{
					{
						Args: []string{"version"},
						Stdout: "Terraform v0.12.0\n",
					},
					{
						Args: []string{"providers", "schema", "-json"},
						Stdout: "{\"format_version\":\"0.1\"}\n",
					},
				},
			},
		})
		md := &discovery.MockDiscovery{Path: "tf-mock"}
		rm.tfDiscoFunc = md.LookPath
		return rm, rm.init(ctx, dir)
	}
	return rmm
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}
