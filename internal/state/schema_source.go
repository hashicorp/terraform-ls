// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import "fmt"

type schemaSrcSigil struct{}

type SchemaSource interface {
	isSchemaSrcImpl() schemaSrcSigil
	String() string
}

type PreloadedSchemaSource struct {
}

func (PreloadedSchemaSource) isSchemaSrcImpl() schemaSrcSigil {
	return schemaSrcSigil{}
}

func (PreloadedSchemaSource) String() string {
	return "preloaded"
}

type LocalSchemaSource struct {
	ModulePath string
}

func (LocalSchemaSource) isSchemaSrcImpl() schemaSrcSigil {
	return schemaSrcSigil{}
}

func (lss LocalSchemaSource) String() string {
	return fmt.Sprintf("local(%s)", lss.ModulePath)
}
