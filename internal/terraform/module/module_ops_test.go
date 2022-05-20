package module

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/registry"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func TestStateStore_getver(t *testing.T) {

	source, _ := tfaddr.ParseRawModuleSourceRegistry("terraform-aws-modules/vpc/aws")

	c, _ := version.NewConstraint(">= 3.0")
	v := registry.GetVersion(source, c)
	if v == nil {
		t.Fatal("should not be nil")
	}
	fmt.Println(v.String())

	if !v.GreaterThan(version.Must(version.NewVersion("3.0.0"))) {
		t.Fatal("version found should be greater than 3.0 nil")
	}
}
