// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"fmt"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/zclconf/go-cty/cty"
)

var orchestrateContextScopeId = lang.ScopeId("orchestrate_context")

// used in various places
var changesAttributes = map[string]cty.Type{
	"total":  cty.Number,
	"add":    cty.Number,
	"change": cty.Number,
	"import": cty.Number,
	"remove": cty.Number,
	"move":   cty.Number,
	"forget": cty.Number,
	"defer":  cty.Number,
}
var changesType = cty.Object(changesAttributes)

func builtinReferences(record *state.StackRecord) reference.Targets {
	targets := make(reference.Targets, 0)

	if record == nil {
		return targets
	}

	// The ranges of the orchestrate blocks as we have to create targets with these ranges
	// to ensure they are only available within orchestrate blocks
	ranges := make([]hcl.Range, 0)
	for _, rule := range record.Meta.OrchestrationRules {
		ranges = append(ranges, rule.Range)
	}

	// The names of the existing components in the stack
	// We use this to offer completions for the component_changes map
	componentNames := make([]string, 0)
	for name := range record.Meta.Components {
		componentNames = append(componentNames, name)
	}

	for _, rng := range ranges {
		// create the static base targets (like context.operation, context.success, etc.)
		targets = append(targets, baseTargets(rng)...)
		// create the static plan targets (like context.plan.mode, context.plan.applyable, etc.)
		targets = append(targets, staticPlanTargets(rng)...)

		// targets for each component for the component_changes map (like context.plan.component_changes["vpc"].total)
		for _, name := range componentNames {
			addr := lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "plan"},
				lang.AttrStep{Name: "component_changes"},
				lang.IndexStep{Key: cty.StringVal(name)},
			}
			targets = append(targets, changesTargets(addr, rng, &name)...)
		}
	}

	return targets
}

func baseTargets(rng hcl.Range) reference.Targets {
	var diagType = cty.Object(map[string]cty.Type{
		"summary": cty.String,
		"detail":  cty.String,
	})

	return reference.Targets{
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "operation"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The operation. Either \"plan\" or \"apply\""),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "success"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.Bool,
			Description:            lang.Markdown("Whether the operation that triggered the evaluation of this check completed successfully"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "errors"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.Set(diagType),
			Description:            lang.Markdown("A set of diagnostic error message objects"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "warnings"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.Set(diagType),
			Description:            lang.Markdown("A set of diagnostic warning message objects"),
		},
	}
}

// staticPlanTargets returns the targets for the plan context that are not dependent on the component names
func staticPlanTargets(rng hcl.Range) reference.Targets {
	targets := reference.Targets{
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "plan"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type: cty.Object(map[string]cty.Type{
				"mode":              cty.String,
				"applyable":         cty.Bool,
				"changes":           changesType,
				"component_changes": cty.Map(changesType),
				"replans":           cty.Number,
				"deployment":        cty.DynamicPseudoType,
			}),
			Description: lang.Markdown("An object including data about the current plan"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "plan"},
				lang.AttrStep{Name: "mode"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.String,
			Description:            lang.Markdown("The plan mode, one of \"normal\", \"refresh-only\", or \"destroy\""),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "plan"},
				lang.AttrStep{Name: "applyable"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.Bool,
			Description:            lang.Markdown("A boolean, whether or not the plan can be applied"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "plan"},
				lang.AttrStep{Name: "replans"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.Number,
			Description:            lang.Markdown("The number of replans in this plan's sequence, starting at 0"),
		},
		{
			LocalAddr: lang.Address{
				lang.RootStep{Name: "context"},
				lang.AttrStep{Name: "plan"},
				lang.AttrStep{Name: "deployment"},
			},
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   cty.DynamicPseudoType,
			Description:            lang.Markdown("A direct reference to the current deployment. Can be used to compare with deployments blocks, e.g. context.plan.deployment == deployment.production"),
		},
	}
	// utility to add all the changes targets like context.plan.changes.total, context.plan.changes.add, etc.
	targets = append(targets, changesTargets(lang.Address{
		lang.RootStep{Name: "context"},
		lang.AttrStep{Name: "plan"},
		lang.AttrStep{Name: "changes"},
	}, rng, nil)...)
	return targets
}

func changesTargets(address lang.Address, rng hcl.Range, componentName *string) reference.Targets {
	descriptionAppendix := "for all components" // default
	if componentName != nil {
		descriptionAppendix = fmt.Sprintf("for the component \"%s\"", *componentName)
	}

	nestedTargets := make(reference.Targets, 0)
	for key, typ := range changesAttributes {
		a := append(address.Copy(), lang.AttrStep{Name: key})
		nestedTargets = append(nestedTargets, reference.Target{
			Name:                   key,
			LocalAddr:              a,
			TargetableFromRangePtr: rng.Ptr(),
			ScopeId:                orchestrateContextScopeId,
			Type:                   typ,
			Description:            lang.Markdown(fmt.Sprintf("The number of %s changes %s", key, descriptionAppendix)),
		})
	}

	return append(nestedTargets, reference.Target{
		LocalAddr:              address,
		TargetableFromRangePtr: rng.Ptr(),
		Type:                   changesType,
		Name:                   "changes",
		Description:            lang.Markdown(fmt.Sprintf("The changes that are planned %s", descriptionAppendix)),
	})
}
