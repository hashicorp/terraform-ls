package schema

import (
	"context"

	tfjson "github.com/hashicorp/terraform-json"
)

type SchemaProvider interface {
	ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error)
	SetWorkdir(string)
}
