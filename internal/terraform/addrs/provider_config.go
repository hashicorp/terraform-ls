package addrs

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// LocalProviderConfig is the address of a provider configuration from the
// perspective of references in a particular module.
//
// Finding the corresponding AbsProviderConfig will require looking up the
// LocalName in the providers table in the module's configuration; there is
// no syntax-only translation between these types.
type LocalProviderConfig struct {
	LocalName string

	// If not empty, Alias identifies which non-default (aliased) provider
	// configuration this address refers to.
	Alias string
}

func (pc LocalProviderConfig) String() string {
	if pc.LocalName == "" {
		// Should never happen; always indicates a bug
		return "provider.<invalid>"
	}

	if pc.Alias != "" {
		return fmt.Sprintf("provider.%s.%s", pc.LocalName, pc.Alias)
	}

	return "provider." + pc.LocalName
}

// StringCompact is an alternative to String that returns the form that can
// be parsed by ParseProviderConfigCompact, without the "provider." prefix.
func (pc LocalProviderConfig) StringCompact() string {
	if pc.Alias != "" {
		return fmt.Sprintf("%s.%s", pc.LocalName, pc.Alias)
	}
	return pc.LocalName
}

// ParseProviderConfigCompact parses the given absolute traversal as a relative
// provider address in compact form. The following are examples of traversals
// that can be successfully parsed as compact relative provider configuration
// addresses:
//
//     aws
//     aws.foo
//
// This function will panic if given a relative traversal.
//
// If the returned diagnostics contains errors then the result value is invalid
// and must not be used.
func ParseProviderConfigCompact(traversal hcl.Traversal) (LocalProviderConfig, error) {
	var errs *multierror.Error

	if len(traversal) == 0 {
		return LocalProviderConfig{}, nil
	}

	ret := LocalProviderConfig{
		LocalName: traversal.RootName(),
	}

	if len(traversal) < 2 {
		// Just a type name, then.
		return ret, errs.ErrorOrNil()
	}

	aliasStep := traversal[1]
	switch ts := aliasStep.(type) {
	case hcl.TraverseAttr:
		ret.Alias = ts.Name
		return ret, errs.ErrorOrNil()
	default:
		errs = multierror.Append(&ParserError{
			Summary: "Invalid provider configuration address",
			Detail:  "The provider type name must either stand alone or be followed by an alias name separated with a dot.",
		})
	}

	if len(traversal) > 2 {
		errs = multierror.Append(&ParserError{
			Summary: "Invalid provider configuration address",
			Detail:  "Extraneous extra operators after provider configuration address.",
		})
	}

	return ret, errs.ErrorOrNil()
}

// ParseProviderConfigCompactStr is a helper wrapper around ParseProviderConfigCompact
// that takes a string and parses it with the HCL native syntax traversal parser
// before interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a reference string is coming from a source that should be identified in
// error messages then the caller should instead parse it directly using a
// suitable function from the HCL API and pass the traversal itself to
// ParseProviderConfigCompact.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// then the returned address is invalid.
func ParseProviderConfigCompactStr(str string) (LocalProviderConfig, error) {
	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	if parseDiags.HasErrors() {
		return LocalProviderConfig{}, parseDiags
	}

	return ParseProviderConfigCompact(traversal)
}
