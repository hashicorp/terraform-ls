module github.com/hashicorp/terraform-ls

go 1.13

require (
	github.com/apparentlymart/go-textseg v1.0.0
	github.com/creachadair/jrpc2 v0.10.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gammazero/workerpool v1.0.0
	github.com/google/go-cmp v0.5.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/hcl-lang v0.0.0-20201116081236-948e43712a65
	github.com/hashicorp/hcl/v2 v2.6.0
	github.com/hashicorp/terraform-exec v0.11.1-0.20201007122305-ea2094d52cb5
	github.com/hashicorp/terraform-json v0.6.0
	github.com/hashicorp/terraform-schema v0.0.0-20201110191417-e2e5d08913c4
	github.com/mh-cbon/go-fmt-fail v0.0.0-20160815164508-67765b3fbcb5
	github.com/mitchellh/cli v1.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.3.2
	github.com/pmezard/go-difflib v1.0.0
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546
	github.com/sourcegraph/go-lsp v0.0.0-20200117082640-b19bb38222e2
	github.com/spf13/afero v1.3.2
	github.com/stretchr/testify v1.4.0
	github.com/vektra/mockery/v2 v2.3.0
)

replace github.com/sourcegraph/go-lsp => github.com/radeksimko/go-lsp v0.1.0
