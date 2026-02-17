// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
)

type StateReader interface {
	PolicyTestRecordByPath(path string) (*state.PolicyTestRecord, error)
	List() ([]*state.PolicyTestRecord, error)
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
			LanguageID: ilsp.PolicyTest.String(),
		})
	}

	return paths
}

// PathContext returns a PathContext for the given path based on the language ID.
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	policytest, err := pr.StateReader.PolicyTestRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}
	return policytestPathContext(policytest, CombinedReader{
		StateReader: pr.StateReader,
		RootReader:  pr.RootReader,
	})
}
