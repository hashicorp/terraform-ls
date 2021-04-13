package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/decoder"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func DecoderForModule(ctx context.Context, mod module.Module) (*decoder.Decoder, error) {
	d := decoder.NewDecoder()
	d.SetUtmSource("terraform-ls")
	d.UseUtmContent(true)

	clientName, ok := lsctx.ClientName(ctx)
	if ok {
		d.SetUtmMedium(clientName)
	}

	for name, f := range mod.ParsedFiles {
		err := d.LoadFile(name, f)
		if err != nil {
			return nil, fmt.Errorf("failed to load a file: %w", err)
		}
	}

	return d, nil
}
