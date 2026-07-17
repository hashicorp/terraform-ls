// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
)

type StateReader interface {
	PolicyRecordByPath(path string) (*state.PolicyRecord, error)
	List() ([]*state.PolicyRecord, error)
}

type RootReader interface {
	TerraformVersion(path string) *version.Version
}

type CombinedReader struct {
	RootReader
	StateReader
}

type PathReader struct {
	RootReader  RootReader
	StateReader StateReader
}

var _ decoder.PathReader = &PathReader{}

func (pr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	records, err := pr.StateReader.List()
	if err != nil {
		return paths
	}

	for _, record := range records {
		paths = append(paths, lang.Path{
			Path:       record.Path(),
			LanguageID: ilsp.Policy.String(),
		})
	}

	return paths
}

// PathContext returns a PathContext for the given path based on the language ID.
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	policy, err := pr.StateReader.PolicyRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}
	return policyPathContext(policy, CombinedReader{
		StateReader: pr.StateReader,
		RootReader:  pr.RootReader,
	})
}
