// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"bytes"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/zclconf/go-cty/cty"
)

func referencesForModule(mod *state.ModuleRecord, stateReader CombinedReader) reference.Targets {
	modPath := mod.Path()
	resolvedVersion := tfschema.ResolveVersion(stateReader.TerraformVersion(modPath), mod.Meta.CoreRequirements)

	targets := tfschema.BuiltinReferencesForVersion(resolvedVersion, modPath)
	targets = append(targets, variableTargets(mod.Meta.Variables)...)
	// For traversals that include index steps (e.g. var.foo[0].bar), we need
	// explicit targets with the matching IndexStep key, otherwise reference
	// completion can't match the prefix.
	targets = append(targets, variableIndexTargets(mod)...)

	return targets
}

func variableTargets(variables map[string]tfmod.Variable) reference.Targets {
	var targets reference.Targets
	for name, v := range variables {
		addr := lang.Address{
			lang.RootStep{Name: "var"},
			lang.AttrStep{Name: name},
		}
		targets = append(targets, subTypeTargets(addr, v.Type, 0)...)
	}
	return targets
}

const maxDeepReferenceDepth = 10

func subTypeTargets(addr lang.Address, typ cty.Type, depth int) reference.Targets {
	if depth >= maxDeepReferenceDepth {
		return nil
	}

	var targets reference.Targets
	if typ.IsObjectType() {
		for attrName, attrType := range typ.AttributeTypes() {
			attrAddr := append(addr.Copy(), lang.AttrStep{Name: attrName})

			targets = append(targets, reference.Target{
				Name:      attrName,
				LocalAddr: attrAddr,
				Type:      attrType,
			})
			targets = append(targets, subTypeTargets(attrAddr, attrType, depth+1)...)
		}
	}
	return targets
}

func variableIndexTargets(mod *state.ModuleRecord) reference.Targets {
	if mod == nil {
		return nil
	}
	if len(mod.Meta.Variables) == 0 {
		return nil
	}
	if len(mod.ParsedModuleFiles) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var targets reference.Targets

	for filename, f := range mod.ParsedModuleFiles {
		if f == nil || len(f.Bytes) == 0 {
			continue
		}
		for _, addr := range extractVarTraversals(string(filename), f.Bytes) {
			if len(addr) < 2 {
				continue
			}
			root, ok := addr[0].(lang.RootStep)
			if !ok || root.Name != "var" {
				continue
			}
			nameStep, ok := addr[1].(lang.AttrStep)
			if !ok {
				continue
			}
			v, ok := mod.Meta.Variables[nameStep.Name]
			if !ok {
				continue
			}

			for _, t := range indexTargetsForTraversal(lang.Address{root, nameStep}, v.Type, addr[2:], 0) {
				k := t.LocalAddr.String()
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
				targets = append(targets, t)
			}
		}
	}

	return targets
}

func extractVarTraversals(filename string, src []byte) []lang.Address {
	needle := []byte("var.")
	addrs := make([]lang.Address, 0)

	// Best-effort extraction: find likely traversal substrings and parse them.
	// We only need to discover literal index keys like [0] or ["database"].
	for i := 0; i < len(src); {
		j := bytes.Index(src[i:], needle)
		if j < 0 {
			break
		}
		start := i + j
		end := scanTraversalEnd(src, start, 300)
		seg := src[start:end]
		if len(seg) == 0 {
			i = start + 1
			continue
		}

		if addr, ok := parseTraversalAddrBestEffort(filename, seg); ok {
			addrs = append(addrs, addr)
		}

		i = start + 1
	}

	return addrs
}

func parseTraversalAddrBestEffort(filename string, src []byte) (lang.Address, bool) {
	// Completion often happens on incomplete expressions like "var.foo.".
	// Try to trim a small set of trailing characters until the traversal parses.
	trimmed := src
	for len(trimmed) > 0 {
		trav, diags := hclsyntax.ParseTraversalAbs(trimmed, filename, hcl.Pos{Line: 1, Column: 1, Byte: 0})
		if !diags.HasErrors() {
			addr, err := lang.TraversalToAddress(trav)
			if err == nil && len(addr) > 0 {
				return addr, true
			}
			return nil, false
		}

		last := trimmed[len(trimmed)-1]
		switch last {
		case '.', '[', '"':
			trimmed = trimmed[:len(trimmed)-1]
			continue
		default:
			return nil, false
		}
	}

	return nil, false
}

func scanTraversalEnd(src []byte, start int, maxLen int) int {
	end := start
	max := start + maxLen
	if max > len(src) {
		max = len(src)
	}

	inStr := false
	escaped := false
	for end < max {
		b := src[end]
		if inStr {
			end++
			if escaped {
				escaped = false
				continue
			}
			if b == '\\' {
				escaped = true
				continue
			}
			if b == '"' {
				inStr = false
			}
			continue
		}

		if b == '"' {
			inStr = true
			end++
			continue
		}

		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '.' || b == '[' || b == ']' {
			end++
			continue
		}

		break
	}

	return end
}

func indexTargetsForTraversal(baseAddr lang.Address, baseType cty.Type, steps []lang.AddressStep, depth int) reference.Targets {
	if depth >= maxDeepReferenceDepth {
		return nil
	}

	var targets reference.Targets
	curAddr := baseAddr.Copy()
	curType := baseType

	for _, st := range steps {
		switch s := st.(type) {
		case lang.AttrStep:
			curAddr = append(curAddr, s)
			curType = attrType(curType, s.Name)
		case lang.IndexStep:
			curAddr = append(curAddr, s)
			elemType := indexElemType(curType, s.Key)
			targets = append(targets, subTypeTargets(curAddr, elemType, depth+1)...)
			curType = elemType
		default:
			return targets
		}
	}

	// Also expand the final type after applying all steps, so that a traversal
	// like "var.foo[0].bar." produces candidates from "bar".
	targets = append(targets, subTypeTargets(curAddr, curType, depth+1)...)

	return targets
}

func attrType(t cty.Type, name string) cty.Type {
	if t.IsObjectType() {
		ats := t.AttributeTypes()
		if at, ok := ats[name]; ok {
			return at
		}
		return cty.DynamicPseudoType
	}
	return cty.DynamicPseudoType
}

func indexElemType(t cty.Type, key cty.Value) cty.Type {
	switch {
	case t.IsListType() || t.IsSetType():
		return t.ElementType()
	case t.IsMapType():
		return t.ElementType()
	case t.IsTupleType():
		if key.Type() == cty.Number {
			f := key.AsBigFloat()
			idx, _ := f.Int64()
			elems := t.TupleElementTypes()
			if idx >= 0 && int(idx) < len(elems) {
				return elems[int(idx)]
			}
		}
		return cty.DynamicPseudoType
	case t.IsObjectType() && key.Type() == cty.String:
		ats := t.AttributeTypes()
		if at, ok := ats[key.AsString()]; ok {
			return at
		}
		return cty.DynamicPseudoType
	default:
		return cty.DynamicPseudoType
	}
}
