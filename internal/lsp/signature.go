package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func ToSignatureHelp(signature *lang.FunctionSignature) *lsp.SignatureHelp {
	if signature == nil {
		return nil
	}

	parameters := make([]lsp.ParameterInformation, 0)
	for _, p := range signature.Parameters {
		parameters = append(parameters, lsp.ParameterInformation{
			Label:         p.Name,
			Documentation: mdplain.Clean(p.Description.Value),
		})
	}

	return &lsp.SignatureHelp{
		Signatures: []lsp.SignatureInformation{
			{
				Label:           signature.Name,
				Documentation:   mdplain.Clean(signature.Description.Value),
				Parameters:      parameters,
				ActiveParameter: signature.ActiveParameter,
			},
		},
		ActiveSignature: 0,
	}
}
