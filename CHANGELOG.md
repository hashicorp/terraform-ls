## 0.2.1 (Unreleased)

BUG FIXES:

 - context: Refactor and fix duplicate key ([#86](https://github.com/hashicorp/terraform-ls/pull/86))

INTERNAL:

 - build: Sign archives checksum ([#99](https://github.com/hashicorp/terraform-ls/pull/99))
 - build: Publish artifacts to releases.hashicorp.com ([#102](https://github.com/hashicorp/terraform-ls/pull/102))

## 0.2.0 (7 May 2020)

FEATURES:

 - Add support for formatting (via `terraform fmt`) ([#51](https://github.com/hashicorp/terraform-ls/pull/51))
 - Add support for completing labels ([#58](https://github.com/hashicorp/terraform-ls/pull/58))

BUG FIXES:

 - Fix URI parsing for Windows paths ([#73](https://github.com/hashicorp/terraform-ls/pull/73))
 - terraform/exec: Make server work under non-admin users on Windows ([#78](https://github.com/hashicorp/terraform-ls/pull/78))

INTERNAL:

 - MacOS and Windows binaries are now signed ([#48](https://github.com/hashicorp/terraform-ls/pull/46))
 - Use Go 1.14.1 (previously `1.13.8`) ([#46](https://github.com/hashicorp/terraform-ls/pull/46))

## 0.1.0 (25 March 2020)

Initial release

FEATURES:

 - Basic text synchronization with client (`didOpen`, `didClose`, `didChange`)
 - Basic block body completion support for attributes and nested blocks
 - Support for standard stdio transport
 - Support for TCP transport (useful for debugging, or reducing the number of LS instances running)
