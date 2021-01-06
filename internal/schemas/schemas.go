// +build !preloadschema

package schemas

import tfjson "github.com/hashicorp/terraform-json"

func PreloadedProviderSchemas() (*tfjson.ProviderSchemas, VersionOutput, error) {
	return nil, VersionOutput{}, nil
}
