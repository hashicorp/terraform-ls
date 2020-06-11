package rootmodule

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestRootModuleManager_RootModuleByPath(t *testing.T) {
	rmm := testRootModuleManager(t)

	direct, unrelated, dirbased := testRootModule(t), testRootModule(t), testRootModule(t)
	rmm.rms = map[string]*rootModule{
		"direct":    direct,
		"unrelated": unrelated,
		"dirbased":  dirbased,
	}

	w1, err := rmm.RootModuleByPath("direct")
	if err != nil {
		t.Fatal(err)
	}
	if direct != w1 {
		t.Fatalf("unexpected root module found: %p, expected: %p", w1, direct)
	}

	w2, err := rmm.RootModuleByPath(filepath.Join("dirbased", ".terraform", "plugins", "selections.json"))
	if err != nil {
		t.Fatal(err)
	}
	if dirbased != w2 {
		t.Fatalf("unexpected root module found: %p, expected: %p", w2, dirbased)
	}
}

func TestRootModuleManager_RootModuleByPath_moduleRefs(t *testing.T) {
	rmm := testRootModuleManager(t)
	direct, unrelated, modbased := testRootModule(t), testRootModule(t), testRootModule(t)

	mm, err := parseModuleManifest([]byte(`{
    "Modules": [
        {
            "Key": "local.deep-inside",
            "Source": "../../another-one",
            "Dir": "another-one"
        },
        {
            "Key": "web_server_sg",
            "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
            "Version": "3.10.0",
            "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/http-80"
        },
        {
            "Key": "web_server_sg.sg",
            "Source": "../../",
            "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0"
        },
        {
            "Key": "",
            "Source": "",
            "Dir": "."
        },
        {
            "Key": "local",
            "Source": "./nested/path",
            "Dir": "nested/path"
        }
    ]
}`))
	if err != nil {
		t.Fatal(err)
	}
	mm.rootDir = "newroot"
	modbased.moduleManifest = mm

	rmm.rms = map[string]*rootModule{
		"direct":      direct,
		"unrelated":   unrelated,
		"modulebased": modbased,
	}

	t.Run("dir-path", func(t *testing.T) {
		w, err := rmm.RootModuleByPath(filepath.Join("newroot", "nested", "path"))
		if err != nil {
			t.Fatal(err)
		}
		if modbased != w {
			t.Fatalf("unexpected root module found: %p, expected: %p", w, modbased)
		}
	})
	t.Run("file-path", func(t *testing.T) {
		_, err := rmm.RootModuleByPath(filepath.Join("newroot", "nested", "path", "file.tf"))
		if err == nil {
			t.Fatal("expected file-based lookup to fail")
		}
	})
}

func testRootModuleManager(t *testing.T) *rootModuleManager {
	rmm := newRootModuleManager(context.Background())
	rmm.logger = testLogger()
	return rmm
}

func testRootModule(t *testing.T) *rootModule {
	w := newRootModule(context.Background())
	w.logger = testLogger()
	return w
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}
