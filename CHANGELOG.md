## 0.7.0 (2 September 2020)

FEATURES:

 - Document Symbol support ([#265](https://github.com/hashicorp/terraform-ls/pull/265))

## 0.6.1 (18 August 2020)

BUG FIXES:

 - Reduce logging of module loading ([#259](https://github.com/hashicorp/terraform-ls/pull/259))
 - Update jrpc2 to fix cancelRequest deadlock ([#260](https://github.com/hashicorp/terraform-ls/pull/260))

## 0.6.0 (10 August 2020)

FEATURES:

 - New command: `inspect-module` to help debugging root module discovery issues ([#231](https://github.com/hashicorp/terraform-ls/pull/231))

ENHANCEMENTS:

 - Support 0.13 provider identities ([#255](https://github.com/hashicorp/terraform-ls/pull/255))
 - settings: Support relative paths to root modules ([#246](https://github.com/hashicorp/terraform-ls/pull/246))
 - settings: Expand `~` in root module paths ([#247](https://github.com/hashicorp/terraform-ls/pull/247))
 - settings: Add support for `excludeModulePaths` ([#251](https://github.com/hashicorp/terraform-ls/pull/251))
 - handlers/initialize: Skip invalid root module paths ([#248](https://github.com/hashicorp/terraform-ls/pull/248))
 - Cap parallel root module loading (to reduce CPU usage) ([#256](https://github.com/hashicorp/terraform-ls/pull/256))

INTERNAL:

 - internal/filesystem: Integrate spf13/afero ([#249](https://github.com/hashicorp/terraform-ls/pull/249))
 - deps: Bump creachadair/jrpc2 to latest (0.10.0) ([#253](https://github.com/hashicorp/terraform-ls/pull/253))

## 0.5.4 (22 July 2020)

BUG FIXES:

 - terraform/schema: Make schema storage version-aware (0.13 compatible) ([#243](https://github.com/hashicorp/terraform-ls/pull/243))

INTERNAL:

 - Improve root module discovery error handling ([#244](https://github.com/hashicorp/terraform-ls/pull/244))

## 0.5.3 (21 July 2020)

BUG FIXES:

 - fix: Append EOF instead of newline (prevent CPU spike) ([#239](https://github.com/hashicorp/terraform-ls/pull/239))

## 0.5.2 (16 July 2020)

BUG FIXES:

 - fix: Prevent parsing invalid tokens which would cause CPU spike ([#236](https://github.com/hashicorp/terraform-ls/pull/236))

INTERNAL:

 - rootmodule: log errors after loading is finished ([#229](https://github.com/hashicorp/terraform-ls/pull/229))

## 0.5.1 (10 July 2020)

BUG FIXES:

 - Fixes bug which broke schema obtaining due to `-no-color` at unsupported position ([#227](https://github.com/hashicorp/terraform-ls/pull/227))

## 0.5.0 (10 July 2020)

ENHANCEMENTS:

 - Introduce CPU & memory profiling ([#223](https://github.com/hashicorp/terraform-ls/pull/223))
 - Pass `-no-color` to terraform ([#208](https://github.com/hashicorp/terraform-ls/pull/208))
 - settings: Make root modules configurable ([#198](https://github.com/hashicorp/terraform-ls/pull/198))

BUG FIXES:

 - terraform/rootmodule: Make walker async by default ([#196](https://github.com/hashicorp/terraform-ls/pull/196))
 - refactor: asynchronous loading of root module parts ([#219](https://github.com/hashicorp/terraform-ls/pull/219))
 - Enable formatting for older Terraform versions (<0.12) ([#219](https://github.com/hashicorp/terraform-ls/pull/219))
 - Gate formatting capability on v0.7.7+ ([#220](https://github.com/hashicorp/terraform-ls/pull/220))

## 0.4.1 (3 July 2020)

BUG FIXES:

 - Make volume comparison case-insensitive on Windows ([#199](https://github.com/hashicorp/terraform-ls/pull/199))

## 0.4.0 (25 June 2020)

FEATURES:

 - Walk hierarchy to add root modules ([#176](https://github.com/hashicorp/terraform-ls/pull/176))

ENHANCEMENTS:

 - terraform: Introduce experimental support for 0.13 version ([#149](https://github.com/hashicorp/terraform-ls/pull/149))
 - Treat schema availability as not essential ([#171](https://github.com/hashicorp/terraform-ls/pull/171))
 - Make formatting work regardless of initialization state ([#178](https://github.com/hashicorp/terraform-ls/pull/178))

BUG FIXES:

 - fix detection of single file during initialization ([#172](https://github.com/hashicorp/terraform-ls/pull/172))

## 0.3.2 (5 June 2020)

BUG FIXES:

 - fix: os.Environ() returns KEY=val, not just keys (fix of a bug that was introduced in 0.3.1) ([#143](https://github.com/hashicorp/terraform-ls/pull/143))

## 0.3.1 (5 June 2020)

BUG FIXES:

 - terraform/exec: Pass through all environment variables ([#139](https://github.com/hashicorp/terraform-ls/pull/139))

## 0.3.0 (4 June 2020)

FEATURES:

 - textDocument/complete: Complete first level keywords ([#104](https://github.com/hashicorp/terraform-ls/pull/104))
 - Add ability to specify path to Terraform binary ([#109](https://github.com/hashicorp/terraform-ls/pull/109))
 - Make Terraform exec timeout configurable ([#134](https://github.com/hashicorp/terraform-ls/pull/134))

ENHANCEMENTS:

 - Improve UX of completion items ([#115](https://github.com/hashicorp/terraform-ls/pull/115))
 - Add support for autocomplete based on a prefix ([#119](https://github.com/hashicorp/terraform-ls/pull/119))
 - textDocument/complete: Use isIncomplete for >100 items ([#132](https://github.com/hashicorp/terraform-ls/pull/132))
 - textDocument/complete: Pass TextEdit instead of static text ([#133](https://github.com/hashicorp/terraform-ls/pull/133))

INTERNAL:

 - refactoring(parser): Pass around tokens instead of blocks ([#125](https://github.com/hashicorp/terraform-ls/pull/125))
 - langserver: Make requests sequential ([#120](https://github.com/hashicorp/terraform-ls/pull/120))
 - Support partial updates ([#103](https://github.com/hashicorp/terraform-ls/pull/103))
 - Support simplified building ([#98](https://github.com/hashicorp/terraform-ls/pull/98))

## 0.2.1 (19 May 2020)

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
