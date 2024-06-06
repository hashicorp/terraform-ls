// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
)

type PathReaderMap map[string]decoder.PathReader

// GlobalPathReader is a PathReader that delegates language specific PathReaders
// that usually come from features.
type GlobalPathReader struct {
	PathReaderMap PathReaderMap
}

var _ decoder.PathReader = &GlobalPathReader{}

func (mr *GlobalPathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	for _, feature := range mr.PathReaderMap {
		paths = append(paths, feature.Paths(ctx)...)
	}

	return paths
}

func (mr *GlobalPathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	if feature, ok := mr.PathReaderMap[path.LanguageID]; ok {
		return feature.PathContext(path)
	}

	return nil, fmt.Errorf("no feature found for language %s", path.LanguageID)
}
