// +build !release

package schemas

import tfjson "github.com/hashicorp/terraform-json"

func PreloadedProviderSchemas() (*tfjson.ProviderSchemas, Version, error) {
	return nil, Version{}, nil
}
