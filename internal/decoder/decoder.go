package decoder

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func DecoderForModule(ctx context.Context, mod module.Module) (*decoder.Decoder, error) {
	d := decoder.NewDecoder()

	d.SetReferenceTargetReader(func() lang.ReferenceTargets {
		return mod.RefTargets
	})

	d.SetReferenceOriginReader(func() lang.ReferenceOrigins {
		return mod.RefOrigins
	})

	d.SetUtmSource("terraform-ls")
	d.UseUtmContent(true)

	clientName, ok := lsctx.ClientName(ctx)
	if ok {
		d.SetUtmMedium(clientName)
	}

	for name, f := range mod.ParsedModuleFiles {
		err := d.LoadFile(name.String(), f)
		if err != nil {
			// skip unreadable files
			continue
		}
	}

	return d, nil
}

func DecoderForVariables(varsFiles ast.VarsFiles) (*decoder.Decoder, error) {
	d := decoder.NewDecoder()

	for name, f := range varsFiles {
		err := d.LoadFile(name.String(), f)
		if err != nil {
			// skip unreadable files
			continue
		}
	}

	return d, nil
}
