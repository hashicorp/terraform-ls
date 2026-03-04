// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/policytest/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// DecodeReferenceTargets collects reference targets,
// using previously parsed AST (via [ParsePolicyTestConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that variable block between certain LOC
// can be referred to as var.foobar. This is useful e.g. during completion,
// go-to-definition or go-to-references.
func DecodeReferenceTargets(ctx context.Context, policytestStore *state.PolicyTestStore, rootFeature fdecoder.RootReader, policytestPath string) error {
	policytest, err := policytestStore.PolicyTestRecordByPath(policytestPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if policytest.RefTargetsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policytestPath)}
	}

	err = policytestStore.SetReferenceTargetsState(policytestPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: policytestStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	pd, err := d.Path(lang.Path{
		Path:       policytestPath,
		LanguageID: ilsp.PolicyTest.String(),
	})
	if err != nil {
		return err
	}

	targets := make(reference.Targets, 0)
	policytestTargets, rErr := pd.CollectReferenceTargets()

	targets = append(targets, policytestTargets...)

	sErr := policytestStore.UpdateReferenceTargets(policytestPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

// DecodeReferenceOrigins collects reference origins,
// using previously parsed AST (via [ParsePolicyTestConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that there is a reference address var.foobar
// at a particular LOC. This can be later matched with targets
// (as obtained via [DecodeReferenceTargets]) during hover or go-to-definition.
func DecodeReferenceOrigins(ctx context.Context, policytestStore *state.PolicyTestStore, rootFeature fdecoder.RootReader, policytestPath string) error {
	policytest, err := policytestStore.PolicyTestRecordByPath(policytestPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if policytest.RefOriginsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policytestPath)}
	}

	err = policytestStore.SetReferenceOriginsState(policytestPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: policytestStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	policytestDecoder, err := d.Path(lang.Path{
		Path:       policytestPath,
		LanguageID: ilsp.PolicyTest.String(),
	})
	if err != nil {
		return err
	}

	origins, rErr := policytestDecoder.CollectReferenceOrigins()

	sErr := policytestStore.UpdateReferenceOrigins(policytestPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
