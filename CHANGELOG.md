## 0.16.0 (Unreleased)

**SECURITY:**

This release is signed with a new GPG key (ID `72D7468F`), unlike all previous releases which were signed with (now revoked) key (ID `348FFC4C`). Old releases were *temporarily* re-signed with the new key, but that key will be removed in coming weeks or months.

[Read more about the related security event HCSEC-2021-12](https://discuss.hashicorp.com/t/hcsec-2021-12-codecov-security-event-and-hashicorp-gpg-key-exposure/23512/2).

Users of the [Terraform VS Code extension](https://github.com/hashicorp/vscode-terraform) will need to upgrade to [`2.10.1`](https://github.com/hashicorp/vscode-terraform/blob/v2.10.1/CHANGELOG.md#2101-2021-04-28) before auto-upgrading to this LS version.


ENHANCEMENTS:

 - Allow effective utilization of multiple schema sources (local or preloaded) via cache ([#454](https://github.com/hashicorp/terraform-ls/issues/454))
 - _"No schema found ..."_ warning removed, as schema is far more likely to be available now ([#454](https://github.com/hashicorp/terraform-ls/issues/454))
 - _"Alternative root modules found ..."_ warning removed ([#454](https://github.com/hashicorp/terraform-ls/issues/454))
 - Further improve support for Terraform 0.15 ([#425](https://github.com/hashicorp/terraform-ls/issues/425))

BUG FIXES:

 - Fix panic caused by partially unknown map keys in configuration ([#447](https://github.com/hashicorp/terraform-ls/issues/447))

## 0.15.0 (12 March 2021)

FEATURES:

 - Add workspace-wide symbol navigation ([#427](https://github.com/hashicorp/terraform-ls/pull/427))

ENHANCEMENTS:

 - textDocument/documentSymbol: Support nested symbols ([#420](https://github.com/hashicorp/terraform-ls/pull/420))
 - Add Go version, OS and architecture to `version` command ([#407](https://github.com/hashicorp/terraform-ls/pull/407))
 - Add initial support for expressions ([#411](https://github.com/hashicorp/terraform-ls/pull/411))
 - Reflect 0.15 schema changes ([#436](https://github.com/hashicorp/terraform-ls/pull/436))

BUILD:

 - Provide Linux packages ([#421](https://github.com/hashicorp/terraform-ls/pull/421))

## 0.14.0 (23 February 2021)

FEATURES:

 - Add links to documentation (Ctrl+click in supported clients + hover) ([#402](https://github.com/hashicorp/terraform-ls/pull/402))

ENHANCEMENTS:

 - Improve messaging when Terraform is not found ([#401](https://github.com/hashicorp/terraform-ls/pull/401))

BUG FIXES:

 - watcher: Refresh versions when plugin lockfile changes ([#403](https://github.com/hashicorp/terraform-ls/pull/403))

BUILD:

 - Provide darwin/arm64 (Apple Silicon) build ([#350](https://github.com/hashicorp/terraform-ls/pull/350))

## 0.13.0 (5 February 2021)

FEATURES:

 - watcher: Detect `terraform init` from scratch ([#385](https://github.com/hashicorp/terraform-ls/pull/385))

ENHANCEMENTS:

 - cmd: Introduce version JSON output ([#386](https://github.com/hashicorp/terraform-ls/pull/386))
 - Utilize CPU better when loading modules ([#391](https://github.com/hashicorp/terraform-ls/pull/391))

BUG FIXES:

 - Fix miscalculated semantic tokens ([#390](https://github.com/hashicorp/terraform-ls/pull/390))

## 0.12.1 (12 January 2021)

BUG FIXES:

 - Print help (and version) to stdout ([#296](https://github.com/hashicorp/terraform-ls/pull/296))
 - Fix broken executable `validate` command ([#373](https://github.com/hashicorp/terraform-ls/pull/373))

## 0.12.0 (6 January 2021)

FEATURES:

 - Implement `textDocument/semanticTokens` (semantic highlighting) ([#331](https://github.com/hashicorp/terraform-ls/pull/331))
 - Implement experimental validate on save feature ([#340](https://github.com/hashicorp/terraform-ls/pull/340))

ENHANCEMENTS:

 - Report progress for validate command ([#336](https://github.com/hashicorp/terraform-ls/pull/336))
 - Report deprecated completion items as such ([#337](https://github.com/hashicorp/terraform-ls/pull/337))
 - Preloaded schemas now include partner providers in addition to official ones ([#341](https://github.com/hashicorp/terraform-ls/pull/341))

NOTES:

 - Only official (legacy) providers will be completed in `provider` block completion. Partner providers currently require corresponding entry in `required_providers` block, read https://github.com/hashicorp/terraform-ls/issues/370 to understand why and how we plan to address this inconvenient behaviour.
 - Preloaded schemas are now being generated at release time (as opposed to being committed to the repo). Therefore availability of these schemas is dependent on particular release process [tracked in this repository](https://github.com/hashicorp/terraform-ls/blob/main/.github/workflows/release.yml). This may interest anyone who does not use the official builds from `releases.hashicorp.com` and has its own build process. Plain `go get` still compiles and runs server correctly, however it won't automatically generate and embed the schemas. ([#341](https://github.com/hashicorp/terraform-ls/pull/341))

INTERNAL:

 - Use Go `1.15.2` (previously `1.14.9`) ([#348](https://github.com/hashicorp/terraform-ls/pull/348))
 - Provide package for linux/arm64 ([#351](https://github.com/hashicorp/terraform-ls/pull/351))

## 0.11.0 (9 December 2020)

ENHANCEMENTS:

 - Ask for init if current folder is empty root module ([#257](https://github.com/hashicorp/terraform-ls/pull/257))
 - Display provider versions in completion/hover detail ([#329](https://github.com/hashicorp/terraform-ls/pull/329))
 - Expose `terraform.validate` as command for language clients ([#323](https://github.com/hashicorp/terraform-ls/pull/323))
 - Expose `terraform.init` as command for language clients ([#325](https://github.com/hashicorp/terraform-ls/pull/325))
 - Add human readable name to `rootmodules` command API ([#332](https://github.com/hashicorp/terraform-ls/pull/332))
 - Expose server version via LSP ([#318](https://github.com/hashicorp/terraform-ls/pull/318))

BUG FIXES:

 - Avoid crashing when no hover data is available for a position ([#320](https://github.com/hashicorp/terraform-ls/pull/320))

INTERNAL:

 - Replace `sourcegraph/go-lsp` with gopls' `internal/lsp/protocol` ([#311](https://github.com/hashicorp/terraform-ls/pull/311))

## 0.10.0 (19 November 2020)

FEATURES:

 - Support module wide diagnostics ([#288](https://github.com/hashicorp/terraform-ls/pull/288))
 - Provide documentation on hover ([#294](https://github.com/hashicorp/terraform-ls/pull/294))

ENHANCEMENTS:

 - Add support for upcoming Terraform v0.14 ([#289](https://github.com/hashicorp/terraform-ls/pull/289))
 - completion: Prompt picking type of provider/data/resource automatically ([#300](https://github.com/hashicorp/terraform-ls/pull/300))
 - completion/hover: Preload official providers to improve UX for uninitialized modules ([#302](https://github.com/hashicorp/terraform-ls/pull/302))

BUG FIXES:

 - textDocument/completion: Fix wrong range computation near EOF ([#298](https://github.com/hashicorp/terraform-ls/pull/298))
 - Avoid ignoring schema for uninitialized module ([#301](https://github.com/hashicorp/terraform-ls/pull/301))
 - fix synchronization issues affecting any clients which support partial updates ([#304](https://github.com/hashicorp/terraform-ls/pull/304))
 - Avoid panic by initing universal schema early ([#307](https://github.com/hashicorp/terraform-ls/pull/307))

INTERNAL:

- Bump jrpc2 (JSON-RPC library) to latest version ([#309](https://github.com/hashicorp/terraform-ls/pull/309))

## 0.9.0 (10 November 2020)

FEATURES:

 - Support for `workspace/executeCommand` with new `rootmodules` inspection command ([#274](https://github.com/hashicorp/terraform-ls/pull/274))
 - Provide version-aware schema for completion of "core" blocks ([#287](https://github.com/hashicorp/terraform-ls/pull/287))
   - `locals`, `module`, `output`, `variable` and `terraform`
   - enrichment of `data`, `provider` and `resource` schemas by meta-arguments, such as `count` or `for_each`

ENHANCEMENTS:

 - Limited completion is available as soon as the server starts and is progressively enhanced as more (core or provider) schema is discovered ([#281](https://github.com/hashicorp/terraform-ls/pull/281))
 - Symbols are available as soon as the server starts ([#281](https://github.com/hashicorp/terraform-ls/pull/281))

BUG FIXES:

 - Prevent command collisions for clients such as VS Code with `commandPrefix` init option ([#279](https://github.com/hashicorp/terraform-ls/pull/279))

INTERNAL:

 - Internal decoder decoupled into `hashicorp/hcl-lang` ([#281](https://github.com/hashicorp/terraform-ls/pull/281))
 - Schema handling decoupled into `hashicorp/terraform-schema` ([#281](https://github.com/hashicorp/terraform-ls/pull/281))

## 0.8.0 (9 October 2020)

FEATURES:

 - HCL diagnostics support ([#269](https://github.com/hashicorp/terraform-ls/pull/269))

BUG FIXES:

 - fix: prevent crash when listing symbols in invalid config ([#273](https://github.com/hashicorp/terraform-ls/pull/273))

INTERNAL:

 - Replace most of `internal/terraform/exec` with [`hashicorp/terraform-exec`](https://github.com/hashicorp/terraform-exec) ([#271](https://github.com/hashicorp/terraform-ls/pull/271))

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
