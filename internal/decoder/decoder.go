package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func DecoderForModule(ctx context.Context, mod module.Module) (*decoder.Decoder, error) {
	d := decoder.NewDecoder()

	d.SetReferenceReader(func() lang.References {
		return mod.References
	})

	d.SetUtmSource("terraform-ls")
	d.UseUtmContent(true)

	clientName, ok := lsctx.ClientName(ctx)
	if ok {
		d.SetUtmMedium(clientName)
	}

	err := loadFiles(d, mod.ParsedModuleFiles)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func DecoderForVariables(mod module.Module) (*decoder.Decoder, error) {
	d := decoder.NewDecoder()

	err := loadFiles(d, mod.ParsedModuleFiles)
	if err != nil {
		return nil, err
	}

	err = loadFiles(d, mod.ParsedVarsFiles)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func loadFiles(d *decoder.Decoder, files map[string]*hcl.File) error {
	for name, f := range files {
		err := d.LoadFile(name, f)
		if err != nil {
			return fmt.Errorf("failed to load a file: %w", err)
		}
	}
	return nil
}
