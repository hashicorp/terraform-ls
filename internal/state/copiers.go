package state

import (
	"reflect"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/mitchellh/copystructure"
	"github.com/zclconf/go-cty/cty"
)

var copiers = map[reflect.Type]copystructure.CopierFunc{
	reflect.TypeOf(cty.NilType):              ctyTypeCopier,
	reflect.TypeOf(cty.Value{}):              ctyValueCopier,
	reflect.TypeOf(version.Version{}):        versionCopier,
	reflect.TypeOf(version.Constraint{}):     constraintCopier,
	reflect.TypeOf(datadir.ModuleManifest{}): modManifestCopier,
	reflect.TypeOf(hcl.File{}):               hclFileCopier,
}

func ctyTypeCopier(v interface{}) (interface{}, error) {
	return v.(cty.Type), nil
}

func ctyValueCopier(v interface{}) (interface{}, error) {
	return v.(cty.Value), nil
}

func versionCopier(v interface{}) (interface{}, error) {
	return v.(version.Version), nil
}

func constraintCopier(v interface{}) (interface{}, error) {
	return v.(version.Constraint), nil
}

func modManifestCopier(v interface{}) (interface{}, error) {
	return v.(datadir.ModuleManifest), nil
}

func hclFileCopier(v interface{}) (interface{}, error) {
	return v.(hcl.File), nil
}
