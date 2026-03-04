// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	tfpolicytest "github.com/hashicorp/terraform-schema/policytest"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	policytestSchema "github.com/hashicorp/terraform-schema/schema/policytest"
)

func schemaForPolicyTest(policytest *state.PolicyTestRecord, stateReader CombinedReader) (*schema.BodySchema, error) {
	version := stateReader.TerraformVersion(policytest.Path())
	if version == nil {
		version = tfschema.LatestAvailableVersion
	}
	schema, err := policytestSchema.CorePolicyTestSchemaForVersion(version)
	if err != nil {
		// this should never happen
		panic(err)
	}
	sm := policytestSchema.NewSchemaMerger(schema)
	sm.SetStateReader(stateReader)

	meta := &tfpolicytest.Meta{
		Path:      policytest.Path(),
		Filenames: policytest.Meta.Filenames,
	}

	return sm.SchemaForPolicyTest(meta)
}
