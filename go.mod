module github.com/hashicorp/terraform-ls

go 1.13

require (
	github.com/apparentlymart/go-textseg v1.0.0
	github.com/creachadair/jrpc2 v0.6.1
	github.com/google/go-cmp v0.4.0
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl/v2 v2.3.0
	github.com/hashicorp/terraform-json v0.4.0
	github.com/mitchellh/cli v1.0.0
	github.com/sourcegraph/go-lsp v0.0.0-20200117082640-b19bb38222e2
	github.com/zclconf/go-cty v1.2.1
	golang.org/x/text v0.3.2
)

replace github.com/sourcegraph/go-lsp => github.com/radeksimko/go-lsp v0.1.0
