## 0.27.0 (Unreleased)

ENHANCEMENTS:

 - Provide (opt-in) custom semantic tokens & modifiers ([#833](https://github.com/hashicorp/terraform-ls/pull/833))

## 0.26.0 (17 March 2022)

ENHANCEMENTS:

 - Introduce go-to-variable from `tfvars` files ([#727](https://github.com/hashicorp/terraform-ls/pull/727))
 - Automatically refresh semantic tokens for more reliable highlighting ([#630](https://github.com/hashicorp/terraform-ls/pull/630))
 - Enhance semantic highlighting of block labels ([#802](https://github.com/hashicorp/terraform-ls/pull/802))
 - Enable completion, hover, go-to-definition/reference etc. for Terraform Registry modules ([#808](https://github.com/hashicorp/terraform-ls/pull/808))
 - Report dependent semantic highlighting modifiers as `defaultLibrary` (instead of `modification`) ([#817](https://github.com/hashicorp/terraform-ls/pull/817))
 - Semantically highlight type declarations in variable `type` ([#827](https://github.com/hashicorp/terraform-ls/pull/827))

BUG FIXES:

 - Address race conditions typically surfaced as "out of range" errors, lack of completion/hover/etc. data or data associated with wrong position within the document ([#782](https://github.com/hashicorp/terraform-ls/pull/782))
 - Fix broken validate on save ([#799](https://github.com/hashicorp/terraform-ls/pull/799))
 - Fix encoding of unknown semantic token types ([#815](https://github.com/hashicorp/terraform-ls/pull/815))
 - Fix missing references for some blocks in a separate config file ([#829](https://github.com/hashicorp/terraform-ls/pull/829))

INTERNAL:

 - Simplify module source detection in favour of faster CI/compilation times ([#783](https://github.com/hashicorp/terraform-ls/pull/783))
 - Store documents in a memdb-backed table ([#771](https://github.com/hashicorp/terraform-ls/pull/771))
 - Refactor job scheduler to use memdb for jobs ([#782](https://github.com/hashicorp/terraform-ls/pull/782))
 - build(deps): bump github.com/creachadair/jrpc2 from 0.35.2 to 0.37.0 ([#774](https://github.com/hashicorp/terraform-ls/pull/774), [#795](https://github.com/hashicorp/terraform-ls/pull/795), [#809](https://github.com/hashicorp/terraform-ls/pull/809))

## 0.25.2 (11 January 2022)

BUG FIXES:

 - fix: avoid sending empty diagnostics ([#756](https://github.com/hashicorp/terraform-ls/pull/756))
 - fix: avoid code lens updates when disabled ([#757](https://github.com/hashicorp/terraform-ls/pull/757))
 - fix: Catch OS agnostic interrupt signal ([#755](https://github.com/hashicorp/terraform-ls/pull/755))
 - fix: Return correct target selection range for definition/declaration ([#759](https://github.com/hashicorp/terraform-ls/pull/759))
 - telemetry: Only send requests if data has changed ([#758](https://github.com/hashicorp/terraform-ls/pull/758))

INTERNAL:

 - Switch to hc-install from tfinstall ([#737](https://github.com/hashicorp/terraform-ls/pull/737))

## 0.25.1 (6 January 2022)

BUG FIXES:

 - Reduce parallelism for background operations to flatten CPU spikes triggered by workspaces with many modules on machines w/ >2 CPUs (which would previously had higher parallelism) ([#752](https://github.com/hashicorp/terraform-ls/pull/752))

INTERNAL:

 - build(deps): bump github.com/creachadair/jrpc2 from 0.32.0 to 0.35.2 ([#748](https://github.com/hashicorp/terraform-ls/pull/748))
 - build(deps): bump github.com/spf13/afero from 1.6.0 to 1.8.0 ([#747](https://github.com/hashicorp/terraform-ls/pull/747), [#754](https://github.com/hashicorp/terraform-ls/pull/754))
 - build(deps): bump github.com/mitchellh/mapstructure from 1.4.2 to 1.4.3 ([#732](https://github.com/hashicorp/terraform-ls/pull/732))
 - build(deps): bump github.com/hashicorp/hcl/v2 from 2.10.1 to 2.11.1 ([#731](https://github.com/hashicorp/terraform-ls/pull/731))

## 0.25.0 (2 December 2021)

ENHANCEMENTS:

 - Introduce `module.providers` command ([#712](https://github.com/hashicorp/terraform-ls/pull/712))
 - Diagnostics for all known modules/files are now published automatically (as opposed to just open files) ([#714](https://github.com/hashicorp/terraform-ls/pull/714))
 - Introduce go-to-variable from module input name ([#700](https://github.com/hashicorp/terraform-ls/pull/700))

NOTES:

 - Diagnostics for non-autoloaded `*.tfvars` are no longer published, see [#715](https://github.com/hashicorp/terraform-ls/issues/715) for more details ([#714](https://github.com/hashicorp/terraform-ls/pull/714))

INTERNAL:

 - build(deps): bump github.com/creachadair/jrpc2 from 0.30.1 to 0.32.0 ([#713](https://github.com/hashicorp/terraform-ls/pull/713), [#728](https://github.com/hashicorp/terraform-ls/pull/728))
 - Avoid duplicate state entries (by avoiding symlink evaluation) ([#699](https://github.com/hashicorp/terraform-ls/pull/699))

## 0.24.0 (10 November 2021)

BREAKING CHANGES:

 - `source.formatAll.terraform-ls` is renamed to `source.formatAll.terraform` to follow other similar existing actions in the wild ([#680](https://github.com/hashicorp/terraform-ls/pull/680))

ENHANCEMENTS:

 - Implement opt-in telemetry (documented in [`docs/telemetry.md`](https://github.com/hashicorp/terraform-ls/blob/v0.24.0/docs/telemetry.md)) ([#681](https://github.com/hashicorp/terraform-ls/pull/681))
 - Provide workspace-wide symbols for variables in `*.tfvars` ([#658](https://github.com/hashicorp/terraform-ls/issues/658))
 - Go-to-definition now highlights just the definition of a block/attribute instead of the whole attribute/block ([#689](https://github.com/hashicorp/terraform-ls/pull/689))
 - Add configuration option allowing to exclude directories from being indexed upon initialization ([#696](https://github.com/hashicorp/terraform-ls/pull/696))
 - Parse `*.tfvars.json` for workspace-wide symbols and diagnostics ([#697](https://github.com/hashicorp/terraform-ls/pull/697))

BUG FIXES:

 - The server announces just a single formatting code action, other actions `source`, `source.fixAll` and `source.formatAll` are removed which helps avoid running the same action multiple times and better follows conventions ([#680](https://github.com/hashicorp/terraform-ls/pull/680))
 - Requesting `Only: []` code actions is now no-op ([#680](https://github.com/hashicorp/terraform-ls/pull/680))
 - Fix indexing of references in dependent modules ([#698](https://github.com/hashicorp/terraform-ls/pull/698))
 - Fix workspace folder removal/addition at runtime ([#707](https://github.com/hashicorp/terraform-ls/pull/707))

INTERNAL:

 - build(deps): bump github.com/creachadair/jrpc2 from 0.28.0 to 0.30.0 ([#683](https://github.com/hashicorp/terraform-ls/pull/683), [#684](https://github.com/hashicorp/terraform-ls/pull/684), [#686](https://github.com/hashicorp/terraform-ls/pull/686))

## 0.23.0 (14 October 2021)

ENHANCEMENTS:

 - Introduce `module.calls` command ([#632](https://github.com/hashicorp/terraform-ls/pull/632))
 - Introduce experimental completion of required fields. You can opt in via [`prefillRequiredFields` option](https://github.com/hashicorp/terraform-ls/blob/v0.23.0/docs/SETTINGS.md#experimentalfeaturesprefillrequiredfields) ([#657](https://github.com/hashicorp/terraform-ls/pull/657))
 - Ignore `.terragrunt-cache` when indexing initialized modules ([#666](https://github.com/hashicorp/terraform-ls/pull/666))
 - Parse `*.tf.json` for references and symbols ([#672](https://github.com/hashicorp/terraform-ls/pull/672))

INTERNAL:

 - build(deps): bump github.com/creachadair/jrpc2 from 0.25.1 to 0.28.0 ([#649](https://github.com/hashicorp/terraform-ls/pull/649), [#650](https://github.com/hashicorp/terraform-ls/pull/650), [#662](https://github.com/hashicorp/terraform-ls/pull/662), [#668](https://github.com/hashicorp/terraform-ls/pull/668), [#676](https://github.com/hashicorp/terraform-ls/pull/676), [#677](https://github.com/hashicorp/terraform-ls/pull/677))
 - build(deps): bump github.com/hashicorp/terraform-exec from 0.14.0 to 0.15.0 ([#664](https://github.com/hashicorp/terraform-ls/pull/664))

## 0.22.0 (16 September 2021)

ENHANCEMENTS:

 - Support standalone (not autoloaded) `*.tfvars` files ([#621](https://github.com/hashicorp/terraform-ls/pull/621))

BUG FIXES:

 - fix: Limit label completion items to 100 (same as limit for completion items in other contexts) ([#628](https://github.com/hashicorp/terraform-ls/pull/628))
 - Recognize references in module block inputs ([#623](https://github.com/hashicorp/terraform-ls/pull/623))

INTERNAL:

 - build(deps): bump github.com/mitchellh/mapstructure from 1.4.1 to 1.4.2 ([#641](https://github.com/hashicorp/terraform-ls/pull/641))
 - build(deps): bump github.com/fsnotify/fsnotify from 1.4.9 to 1.5.1 ([#629](https://github.com/hashicorp/terraform-ls/pull/629))
 - build(deps): bump github.com/creachadair/jrpc2 from 0.20.0 to 0.25.0 ([#631](https://github.com/hashicorp/terraform-ls/pull/631), [#636](https://github.com/hashicorp/terraform-ls/pull/636), [#638](https://github.com/hashicorp/terraform-ls/pull/638), [#640](https://github.com/hashicorp/terraform-ls/pull/640), [#642](https://github.com/hashicorp/terraform-ls/pull/642))

## 0.21.0 (23 August 2021)

DEPRECATIONS:

 - `-tf-exec` (CLI flag) is deprecated in favour of LSP config option [`terraformExecPath`](https://github.com/hashicorp/terraform-ls/blob/v0.21.0/docs/SETTINGS.md#terraformexecpath-string). `-tf-exec` flag will raise warnings in future releases and will be eventually removed. ([#588](https://github.com/hashicorp/terraform-ls/pull/588))
 - `-tf-log-file` (CLI flag) is deprecated in favour of LSP config option [`terraformLogFilePath`](https://github.com/hashicorp/terraform-ls/blob/v0.21.0/docs/SETTINGS.md#terraformlogfilepath-string). `-tf-log-file` flag will raise warnings in future releases and will be eventually removed. ([#619](https://github.com/hashicorp/terraform-ls/pull/619))
 - `tf-exec-timeout` (CLI flag) is deprecated in favour of LSP config option [`terraformExecTimeout`](https://github.com/hashicorp/terraform-ls/blob/v0.21.0/docs/SETTINGS.md#terraformexectimeout-string). `tf-exec-timeout` flag will raise warnings in future releases and will be eventually removed. ([#619](https://github.com/hashicorp/terraform-ls/pull/619))

BUG FIXES:

 - fix: allow multiple variable validation blocks ([#610](https://github.com/hashicorp/terraform-ls/pull/610))
 - fix: avoid crash on missing block label ([#612](https://github.com/hashicorp/terraform-ls/pull/612))
 - fix: avoid crash when `validate` command returns internal error instead of diagnostics ([#588](https://github.com/hashicorp/terraform-ls/pull/588))

ENHANCEMENTS:

 - Always validate URI schema ([#602](https://github.com/hashicorp/terraform-ls/pull/602))
 - Introduce [`terraformExecPath`](https://github.com/hashicorp/terraform-ls/blob/v0.21.0/docs/SETTINGS.md#terraformexecpath-string) as option within `initializationOptions` ([#588](https://github.com/hashicorp/terraform-ls/pull/588))
 - Introduce [`terraformLogFilePath`](https://github.com/hashicorp/terraform-ls/blob/v0.21.0/docs/SETTINGS.md#terraformlogfilepath-string) as option within `initializationOptions` ([#619](https://github.com/hashicorp/terraform-ls/pull/619))
 - Introduce [`terraformExecTimeout`](https://github.com/hashicorp/terraform-ls/blob/v0.21.0/docs/SETTINGS.md#terraformexectimeout-string) as option within `initializationOptions` ([#619](https://github.com/hashicorp/terraform-ls/pull/619))
 - Introduce format on save code action ([#625](https://github.com/hashicorp/terraform-ls/pull/625))

INTERNAL:

 - Update LSP structs to gopls' `0.7.0` ([#608](https://github.com/hashicorp/terraform-ls/pull/608))
 - build(deps): bump github.com/creachadair/jrpc2 from 0.19.1 to 0.20.0 ([#614](https://github.com/hashicorp/terraform-ls/pull/614))
 - build(deps): bump github.com/zclconf/go-cty from 1.9.0 to 1.9.1 ([#624](https://github.com/hashicorp/terraform-ls/pull/624))

## 0.20.1 (3 August 2021)

BUG FIXES:

 - fix: recognize references in common nested expressions ([#596](https://github.com/hashicorp/terraform-ls/pull/596))
 - textDocument/publishDiagnostics: Publish any source-less warnings or errors ([#601](https://github.com/hashicorp/terraform-ls/pull/601))
 - fix: avoid publishing stale 'validate' diagnostics ([#603](https://github.com/hashicorp/terraform-ls/pull/603))
 - fix: avoid crash on highlighting unknown tuple element ([#605](https://github.com/hashicorp/terraform-ls/pull/605))
 - fix: recognize list(object) and set(object) attributes as blocks ([#607](https://github.com/hashicorp/terraform-ls/pull/607))

INTERNAL:

 - build(deps): bump github.com/creachadair/jrpc2 from 0.19.0 to 0.19.1 ([#606](https://github.com/hashicorp/terraform-ls/pull/606))

## 0.20.0 (29 July 2021)

FEATURES:

 - Implement reference count code lens ([#584](https://github.com/hashicorp/terraform-ls/pull/584))

ENHANCEMENTS:

 - Add support for module input completion/hover/highlighting ([#551](https://github.com/hashicorp/terraform-ls/pull/551))
 - Add support for module output reference completion/hover/highlighting ([#593](https://github.com/hashicorp/terraform-ls/pull/593))

BUG FIXES:

 - fix: recognize references in lists and other complex types ([#594](https://github.com/hashicorp/terraform-ls/pull/594))

INTERNAL:

 - build(deps): bump github.com/hashicorp/hcl/v2 from 2.10.0 to 2.10.1 ([#589](https://github.com/hashicorp/terraform-ls/pull/589))

## 0.19.1 (20 July 2021)

BUG FIXES:

 - Fix 'go to references' for resources & data sources ([#587](https://github.com/hashicorp/terraform-ls/pull/587))

INTERNAL:

 - build(deps): bump github.com/creachadair/jrpc2 from 0.17.0 to 0.18.0 ([#550](https://github.com/hashicorp/terraform-ls/pull/550))

## 0.19.0 (8 July 2021)

FEATURES:

 - Go to attribute/block from reference ([#569](https://github.com/hashicorp/terraform-ls/pull/569))
 - Go to references from an attribute or a block ([#572](https://github.com/hashicorp/terraform-ls/pull/572), [#580](https://github.com/hashicorp/terraform-ls/pull/580))

ENHANCEMENTS:

 - Support multiple folders natively ([#502](https://github.com/hashicorp/terraform-ls/pull/502))
 - Make references scope & type aware ([#582](https://github.com/hashicorp/terraform-ls/pull/582))

BUG FIXES:

 - fix: avoid crash on empty file formatting ([#578](https://github.com/hashicorp/terraform-ls/pull/578))

## 0.18.3 (2 July 2021)

BUG FIXES:

 - fix: avoid circular references to list/map/object attributes (which caused high CPU usage on copy) ([#575](https://github.com/hashicorp/terraform-ls/pull/575))

## 0.18.2 (1 July 2021)

ENHANCEMENTS:

 - Provide (less verbose) step-based completion ([#566](https://github.com/hashicorp/terraform-ls/pull/566))

BUG FIXES:

 - Mock out code lens support to avoid errors ([#561](https://github.com/hashicorp/terraform-ls/pull/561))

## 0.18.1 (17 June 2021)

ENHANCEMENTS:

 - Support for references to variables and locals ([#553](https://github.com/hashicorp/terraform-ls/pull/553))
 - tfvars: Infer variable types from default values where not explicitly specified ([#554](https://github.com/hashicorp/terraform-ls/pull/554))

BUG FIXES:

 - Prevent var names from being completed in label ([#555](https://github.com/hashicorp/terraform-ls/pull/555))

## 0.18.0 (10 June 2021)

FEATURES:

 - Add support for `tfvars` (variable files) ([#540](https://github.com/hashicorp/terraform-ls/pull/540))

ENHANCEMENTS:

 - Add support for state backends ([#544](https://github.com/hashicorp/terraform-ls/pull/544))
 - Add support for provisioners ([#542](https://github.com/hashicorp/terraform-ls/pull/542))
 - Support for type declarations (variable `type`) ([#490](https://github.com/hashicorp/terraform-ls/pull/490))
 - Support variable `default` ([#543](https://github.com/hashicorp/terraform-ls/pull/543))

## 0.17.1 (26 May 2021)

BUG FIXES:

 - Reduce CPU usage via custom Copy methods instead reflection ([#513](https://github.com/hashicorp/terraform-ls/pull/513))

## 0.17.0 (20 May 2021)

ENHANCEMENTS:

 - Add support for traversals/references ([#485](https://github.com/hashicorp/terraform-ls/pull/485))
 - Add new `module.callers` (LSP) command & [document all available commands](https://github.com/hashicorp/terraform-ls/blob/41d49b3/docs/commands.md) ([#508](https://github.com/hashicorp/terraform-ls/pull/508))

## 0.16.3 (13 May 2021)

ENHANCEMENTS:

 - Increase request concurrency & make it configurable via `-req-concurrency` flag of `serve` command ([#489](https://github.com/hashicorp/terraform-ls/pull/489))

BUG FIXES:

 - Fix request cancellation ([#314](https://github.com/hashicorp/terraform-ls/issues/314))

## 0.16.2 (11 May 2021)

ENHANCEMENTS:

 - Support templated paths for `-cpuprofile` & `-memprofile` flags of `serve` ([#501](https://github.com/hashicorp/terraform-ls/pull/501))

BUG FIXES:

 - Avoid presenting stale diagnostics after document changes ([#488](https://github.com/hashicorp/terraform-ls/pull/488))

## 0.16.1 (30 April 2021)

BUG FIXES:

 - Prevent crash for legacy provider lookups where configuration is missing `terraform`>`required_providers` block or `source` arguments for providers and Terraform 0.13+ is used ([#481](https://github.com/hashicorp/terraform-ls/pull/481))

## 0.16.0 (29 April 2021)

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
