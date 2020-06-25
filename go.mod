module github.com/hashicorp/terraform-ls

go 1.13

require (
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/apparentlymart/go-textseg v1.0.0
	github.com/creachadair/jrpc2 v0.8.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/google/go-cmp v0.4.0
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl/v2 v2.5.2-0.20200528183353-fa7c453538de
	github.com/hashicorp/terraform-json v0.5.0
	github.com/mitchellh/cli v1.0.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/sourcegraph/go-lsp v0.0.0-20200117082640-b19bb38222e2
	github.com/zclconf/go-cty v1.2.1
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527 // indirect
	golang.org/x/text v0.3.2
)

replace github.com/sourcegraph/go-lsp => github.com/radeksimko/go-lsp v0.1.0
