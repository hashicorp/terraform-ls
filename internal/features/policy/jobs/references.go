// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"strings"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/policy/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfpolicy "github.com/hashicorp/terraform-schema/policy"
)

// DecodeReferenceTargets collects reference targets,
// using previously parsed AST (via [ParsePolicyConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that variable block between certain LOC
// can be referred to as var.foobar. This is useful e.g. during completion,
// go-to-definition or go-to-references.
func DecodeReferenceTargets(ctx context.Context, policyStore *state.PolicyStore, rootFeature fdecoder.RootReader, policyPath string) error {
	policy, err := policyStore.PolicyRecordByPath(policyPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if policy.RefTargetsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policyPath)}
	}

	err = policyStore.SetReferenceTargetsState(policyPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: policyStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	pd, err := d.Path(lang.Path{
		Path:       policyPath,
		LanguageID: ilsp.Policy.String(),
	})
	if err != nil {
		return err
	}
	record, err := policyStore.PolicyRecordByPath(policyPath)
	if err != nil {
		return err
	}

	targets := make(reference.Targets, 0)
	policyTargets, rErr := pd.CollectReferenceTargets()
	builtinTargets := builtinReferences(record)

	configurePolicyScopedLocalTargets(policyTargets, record.Meta.ResourcePolicies, record.Meta.ProviderPolicies, record.Meta.ModulePolicies)
	targets = append(targets, policyTargets...)
	targets = append(targets, builtinTargets...)

	sErr := policyStore.UpdateReferenceTargets(policyPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

// DecodeReferenceOrigins collects reference origins,
// using previously parsed AST (via [ParsePolicyConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that there is a reference address var.foobar
// at a particular LOC. This can be later matched with targets
// (as obtained via [DecodeReferenceTargets]) during hover or go-to-definition.
func DecodeReferenceOrigins(ctx context.Context, policyStore *state.PolicyStore, rootFeature fdecoder.RootReader, policyPath string) error {
	policy, err := policyStore.PolicyRecordByPath(policyPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if policy.RefOriginsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policyPath)}
	}

	err = policyStore.SetReferenceOriginsState(policyPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: policyStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	policyDecoder, err := d.Path(lang.Path{
		Path:       policyPath,
		LanguageID: ilsp.Policy.String(),
	})
	if err != nil {
		return err
	}

	origins, rErr := policyDecoder.CollectReferenceOrigins()

	sErr := policyStore.UpdateReferenceOrigins(policyPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

func configurePolicyScopedLocalTargets(targets reference.Targets, resourcePolicies map[string]tfpolicy.ResourcePolicy, providerPolicies map[string]tfpolicy.ProviderPolicy, modulePolicies map[string]tfpolicy.ModulePolicy) {
	for i := range targets {
		target := &targets[i]
		if strings.HasPrefix(target.Addr.String(), "local.") && (target.ScopeId == lang.ScopeId("resource_policy") || target.ScopeId == lang.ScopeId("provider_policy") || target.ScopeId == lang.ScopeId("module_policy")) {
			// Swap LocalAddr and Addr as locals are locally scoped to a block
			target.LocalAddr = target.Addr.Copy()
			target.Addr = nil

			if target.TargetableFromRangePtr != nil {
				continue
			}

			// For locally scoped local block add it's TargetableFromRangePtr
			for _, rp := range resourcePolicies {
				if target.RangePtr.Overlaps(rp.Range) {
					target.TargetableFromRangePtr = &rp.Range
					break
				}
			}

			for _, pp := range providerPolicies {
				if target.RangePtr.Overlaps(pp.Range) {
					target.TargetableFromRangePtr = &pp.Range
					break
				}
			}

			for _, mp := range modulePolicies {
				if target.RangePtr.Overlaps(mp.Range) {
					target.TargetableFromRangePtr = &mp.Range
					break
				}
			}

		}
		// Recursively modify nested targets
		if len(target.NestedTargets) > 0 {
			configurePolicyScopedLocalTargets(target.NestedTargets, resourcePolicies, providerPolicies, modulePolicies)
		}
	}
}
