## 0.31.2 (11 May 2023)

BUG FIXES:

* Fix crash on prefix completion ([hcl-lang#275](https://github.com/hashicorp/hcl-lang/pull/275))

INTERNAL:

* Remove automated milestone closure commenting ([#1266](https://github.com/hashicorp/terraform-ls/pull/1266))

## 0.31.1 (27 April 2023)

ENHANCEMENTS:

* Cache registry module errors to improve performance in cases of private registry, submodules or other similar situations resulting in module data unavailability ([#1258](https://github.com/hashicorp/terraform-ls/pull/1258))

BUG FIXES:

* Pull in gopls v0.10.0 tsprotocol.go to fix completion labels ([#1256](https://github.com/hashicorp/terraform-ls/pull/1256))

INTERNAL:

* Add PR test for copyright headers ([#1241](https://github.com/hashicorp/terraform-ls/pull/1241))

## 0.31.0 (18 April 2023)

ENHANCEMENTS:

* Add support for nested expressions and functions ([#1237](https://github.com/hashicorp/terraform-ls/pull/1237), [hcl-lang#232](https://github.com/hashicorp/hcl-lang/pull/232), [hcl-lang#203](https://github.com/hashicorp/hcl-lang/pull/203), [hcl-lang#199](https://github.com/hashicorp/hcl-lang/pull/199), [hcl-lang#186](https://github.com/hashicorp/hcl-lang/pull/186), [hcl-lang#185](https://github.com/hashicorp/hcl-lang/pull/185), [hcl-lang#184](https://github.com/hashicorp/hcl-lang/pull/184))
* Add support for function signature help in ([#1077](https://github.com/hashicorp/terraform-ls/pull/1077))
* Fix remote backend tracking in ([#1218](https://github.com/hashicorp/terraform-ls/pull/1218))
* lsp: Recognise new token type for function names in ([#1233](https://github.com/hashicorp/terraform-ls/pull/1233))

INTERNAL:

* Add instructions for Kate editor ([#1200](https://github.com/hashicorp/terraform-ls/pull/1200))
* Add TFC usage detection ([#1208](https://github.com/hashicorp/terraform-ls/pull/1208))

BUG FIXES:

* Reflect `LiteralValue`, `Description` & `IsDeprecated` in completion/hover ([hcl-lang#253](https://github.com/hashicorp/hcl-lang/pull/253))
* Fix crash when completing `LiteralType{Type: cty.Tuple}` ([hcl-lang#255](https://github.com/hashicorp/hcl-lang/pull/255))
* Display `Tuple` hover data on invalid element ([hcl-lang#254](https://github.com/hashicorp/hcl-lang/pull/254))
* Fix collection of implied declared targets of complex types ([hcl-lang#259](https://github.com/hashicorp/hcl-lang/pull/259))
* Collect targets w/ interpolation for `Any` correctly ([hcl-lang#257](https://github.com/hashicorp/hcl-lang/pull/257))

## 0.31.0-beta (6 April 2023)

ENHANCEMENTS:

* Add support for nested expressions and functions ([#1237](https://github.com/hashicorp/terraform-ls/pull/1237), [hcl-lang#232](https://github.com/hashicorp/hcl-lang/pull/232), [hcl-lang#203](https://github.com/hashicorp/hcl-lang/pull/203), [hcl-lang#199](https://github.com/hashicorp/hcl-lang/pull/199), [hcl-lang#186](https://github.com/hashicorp/hcl-lang/pull/186), [hcl-lang#185](https://github.com/hashicorp/hcl-lang/pull/185), [hcl-lang#184](https://github.com/hashicorp/hcl-lang/pull/184), )
* Add support for function signature help in ([#1077](https://github.com/hashicorp/terraform-ls/pull/1077))
* Fix remote backend tracking in ([#1218](https://github.com/hashicorp/terraform-ls/pull/1218))
* lsp: Recognise new token type for function names in ([#1233](https://github.com/hashicorp/terraform-ls/pull/1233))

INTERNAL:

* Add instructions for Kate editor ([#1200](https://github.com/hashicorp/terraform-ls/pull/1200))
* Add TFC usage detection ([#1208](https://github.com/hashicorp/terraform-ls/pull/1208))
* build(deps): bump actions/checkout from 3.3.0 to 3.4.0 ([#1215](https://github.com/hashicorp/terraform-ls/pull/1215))
* build(deps): bump actions/checkout from 3.4.0 to 3.5.0 ([#1228](https://github.com/hashicorp/terraform-ls/pull/1228))
* build(deps): bump actions/setup-go from 3.5.0 to 4.0.0 ([#1214](https://github.com/hashicorp/terraform-ls/pull/1214))
* build(deps): bump actions/stale from 7.0.0 to 8.0.0 ([#1222](https://github.com/hashicorp/terraform-ls/pull/1222))
* build(deps): bump github.com/algolia/algoliasearch-client-go/v3 from 3.26.3 to 3.26.4 ([#1198](https://github.com/hashicorp/terraform-ls/pull/1198))
* build(deps): bump github.com/algolia/algoliasearch-client-go/v3 from 3.26.4 to 3.26.5 ([#1230](https://github.com/hashicorp/terraform-ls/pull/1230))
* build(deps): bump github.com/algolia/algoliasearch-client-go/v3 from 3.26.5 to 3.27.0 ([#1231](https://github.com/hashicorp/terraform-ls/pull/1231))
* build(deps): bump github.com/creachadair/jrpc2 from 0.44.0 to 0.45.0 ([#1213](https://github.com/hashicorp/terraform-ls/pull/1213))
* build(deps): bump github.com/creachadair/jrpc2 from 0.46.0 to 1.0.0 ([#1227](https://github.com/hashicorp/terraform-ls/pull/1227))
* build(deps): bump github.com/creachadair/jrpc2 to v0.46.0 ([#1217](https://github.com/hashicorp/terraform-ls/pull/1217))
* build(deps): bump github.com/hashicorp/hc-install from 0.5.0 to 0.5.1 ([#1232](https://github.com/hashicorp/terraform-ls/pull/1232))
* build(deps): bump github.com/hashicorp/hcl/v2 from 2.16.1 to 2.16.2 ([#1205](https://github.com/hashicorp/terraform-ls/pull/1205))
* build(deps): bump github.com/hashicorp/terraform-exec from 0.18.0 to 0.18.1 ([#1201](https://github.com/hashicorp/terraform-ls/pull/1201))
* build(deps): bump github.com/hashicorp/terraform-json from 0.15.0 to 0.16.0 ([#1206](https://github.com/hashicorp/terraform-ls/pull/1206))
* build(deps): bump github.com/hashicorp/terraform-registry-address from 0.0.0-20220623143253-7d51757b572c to 0.1.0 ([#1196](https://github.com/hashicorp/terraform-ls/pull/1196))
* build(deps): bump github.com/hashicorp/terraform-registry-address from 0.1.0 to 0.2.0 ([#1226](https://github.com/hashicorp/terraform-ls/pull/1226))
* build(deps): bump github.com/otiai10/copy from 1.9.0 to 1.10.0 ([#1236](https://github.com/hashicorp/terraform-ls/pull/1236))
* build(deps): bump github.com/stretchr/testify from 1.8.1 to 1.8.2 ([#1199](https://github.com/hashicorp/terraform-ls/pull/1199))
* build(deps): bump github.com/vektra/mockery/v2 from 2.20.2 to 2.21.1 ([#1202](https://github.com/hashicorp/terraform-ls/pull/1202))
* build(deps): bump github.com/vektra/mockery/v2 from 2.21.1 to 2.21.4 ([#1204](https://github.com/hashicorp/terraform-ls/pull/1204))
* build(deps): bump github.com/vektra/mockery/v2 from 2.21.4 to 2.21.6 ([#1207](https://github.com/hashicorp/terraform-ls/pull/1207))
* build(deps): bump github.com/vektra/mockery/v2 from 2.21.6 to 2.22.1 ([#1209](https://github.com/hashicorp/terraform-ls/pull/1209))
* build(deps): bump github.com/vektra/mockery/v2 from 2.22.1 to 2.23.0 ([#1219](https://github.com/hashicorp/terraform-ls/pull/1219))
* build(deps): bump github.com/vektra/mockery/v2 from 2.23.0 to 2.23.1 ([#1221](https://github.com/hashicorp/terraform-ls/pull/1221))
* build(deps): bump github.com/vektra/mockery/v2 from 2.23.1 to 2.23.2 ([#1235](https://github.com/hashicorp/terraform-ls/pull/1235))
* build(deps): bump github.com/zclconf/go-cty from 1.12.1 to 1.13.0 ([#1197](https://github.com/hashicorp/terraform-ls/pull/1197))
* build(deps): bump github.com/zclconf/go-cty from 1.13.0 to 1.13.1 ([#1216](https://github.com/hashicorp/terraform-ls/pull/1216))
* build(deps): bump golang.org/x/tools from 0.6.0 to 0.7.0 ([#1203](https://github.com/hashicorp/terraform-ls/pull/1203))
* build(deps): Bump hcl-lang & terraform-schema to latest revisions ([#1237](https://github.com/hashicorp/terraform-ls/pull/1237))

## 0.30.3 (22 February 2023)

BUG FIXES:

 - Enable static builds of Linux binaries (again) ([#1193](https://github.com/hashicorp/terraform-ls/pull/1193))

## 0.30.2 (15 February 2023)

NOTES / BREAKING CHANGES:

 - We have changed our release process: all assets continue to be available from the [HashiCorp Releases site](https://releases.hashicorp.com/terraform-ls) and/or via the [Releases API](https://releases.hashicorp.com/docs/api/v1/), not as GitHub Release assets anymore.

ENHANCEMENTS:

 - Parse `optional()` object attribute _default values_ correctly, as introduced in Terraform v1.3 ([terraform-schema#184](https://github.com/hashicorp/terraform-schema/pull/184))
 
BUG FIXES:

 - Ignore inaccessible files (such as emacs backup files) ([terraform-ls#1172](https://github.com/hashicorp/terraform-ls/issues/1067]))
 - Fix crash when parsing JSON files (introduced in 0.30.0) ([hcl-lang#202](https://github.com/hashicorp/hcl-lang/pull/202]))

INTERNAL:

 - Remove `schema.TupleConsExpr` ([hcl-lang#175](https://github.com/hashicorp/hcl-lang/pull/175))
 - internal/schema: Replace `TupleConsExpr` with `SetExpr` ([terraform-schema#169](https://github.com/hashicorp/terraform-schema/pull/169))
 - Use upstreamed HCL typexpr package ([terraform-schema#168](https://github.com/hashicorp/terraform-schema/pull/168))

## 0.30.1 (1 December 2022)

BUG FIXES:

 - Support `dynamic` in the `provisioner` and `provider` blocks ([terraform-schema#165](https://github.com/hashicorp/terraform-schema/pull/165))
 - Fix `dynamic` block `for_each` description ([hcl-lang#164](https://github.com/hashicorp/hcl-lang/pull/164))
 - Avoid completing static block inside a `dynamic` label ([hcl-lang#165](https://github.com/hashicorp/hcl-lang/pull/165))
 - Fix missing hover for `count` and `for_each` expression ([hcl-lang#166](https://github.com/hashicorp/hcl-lang/pull/166))
 - Fix support of deeper nesting of `dynamic` block ([hcl-lang#167](https://github.com/hashicorp/hcl-lang/pull/167))
 - Change `dynamic` block type to default ([hcl-lang#168](https://github.com/hashicorp/hcl-lang/pull/168))

## 0.30.0 (24 November 2022)

ENHANCEMENTS:

 - Support `count.index` references in blocks with `count` for completion, hover documentation and semantic tokens highlighting ([#860](https://github.com/hashicorp/terraform-ls/issues/860), [hcl-lang#160](https://github.com/hashicorp/hcl-lang/pull/160))
 - Support `each.*` references in blocks with `for_each` for completion, hover documentation and semantic tokens highlighting ([#861](https://github.com/hashicorp/terraform-ls/issues/861), [hcl-lang#162](https://github.com/hashicorp/hcl-lang/pull/162))
 - Support `self.*` references in `provisioner`, `connection` and `postcondition` blocks for completion, hover documentation and semantic tokens highlighting ([#859](https://github.com/hashicorp/terraform-ls/issues/859), [hcl-lang#163](https://github.com/hashicorp/hcl-lang/pull/163))
 - `dynamic` block support, including label and content completion ([#530](https://github.com/hashicorp/terraform-ls/issues/530), [hcl-lang#154](https://github.com/hashicorp/hcl-lang/pull/154))
 - Go-to-definition/go-to-references for `count.index`/`count` ([#1093](https://github.com/hashicorp/terraform-ls/issues/1093))
 - Go-to-definition/go-to-references for `each.*`/`for_each` ([#1095](https://github.com/hashicorp/terraform-ls/issues/1095))
 - Go-to-definition/go-to-references for `self.*` in `provisioner`, `connection` and `postcondition` blocks ([#1096](https://github.com/hashicorp/terraform-ls/issues/1096))
 - Remove deprecated backends in Terraform 1.3.0 ([terraform-schema#159](https://github.com/hashicorp/terraform-schema/pull/159))

## 0.29.3 (13 October 2022)

ENHANCEMENTS:

 - schemas: Lazy-load embedded provider schemas ([#1071](https://github.com/hashicorp/terraform-ls/pull/1071))
   - Reduced runtime memory consumption from static ~572MB (representing ~220 providers) to more dynamic depending on providers in use.
     For example, no configuration (no provider requirements) should consume around 10MB, indexed folder w/ `hashicorp/aws` requirement ~70MB.
   - Reduced launch time from ~ 2 seconds to 1-3 ms.

BUG FIXES:

 - fix: Enable IntelliSense for resources & data sources whose name match the provider (e.g. `data`) ([#1072](https://github.com/hashicorp/terraform-ls/pull/1072))
 - state: avoid infinite recursion (surfaced as crash with "goroutine stack exceeds 1000000000-byte limit" message) ([#1084](https://github.com/hashicorp/terraform-ls/pull/1084))
 - decoder: fix race condition in terraform-schema (surfaced as crash with "fatal error: concurrent map read and map write" message) ([#1086](https://github.com/hashicorp/terraform-ls/pull/1086))

## 0.29.2 (7 September 2022)

BUG FIXES:

 - fix: Improve IntelliSense accuracy by tracking provider schema versions (accidentally removed in 0.29.0) ([#1060](https://github.com/hashicorp/terraform-ls/pull/1060))
 - Don't query the Terraform Registry for module sources starting with `.` ([#1062](https://github.com/hashicorp/terraform-ls/pull/1062))
 - fix race condition in schema merging ([terraform-schema#137](https://github.com/hashicorp/terraform-schema/pull/137))

INTERNAL:

 - Use Go 1.19 (previously 1.17) to build the server ([#1046](https://github.com/hashicorp/terraform-ls/pull/1046))

## 0.29.1 (24 August 2022)

ENHANCEMENTS:

 - docs: Add link to post explaining vim plugin installation ([#1044](https://github.com/hashicorp/terraform-ls/pull/1044))

BUG FIXES:

 - goreleaser: Use correct ldflag (versionPrerelease) when compiling LS ([#1043](https://github.com/hashicorp/terraform-ls/pull/1043))
 - Fix panic on obtaining provider schemas ([#1048](https://github.com/hashicorp/terraform-ls/pull/1048))

INTERNAL:

 - cleanup: Remove LogHandler ([#1038](https://github.com/hashicorp/terraform-ls/pull/1038))

## 0.29.0 (11 August 2022)

NOTES / BREAKING CHANGES:

 - settings: `rootModulePaths` option was deprecated and is ignored. Users should instead leverage the workspace LSP API and add the folder to a workspace, if they wish it to be indexed ([#1003](https://github.com/hashicorp/terraform-ls/pull/1003))
 - settings: `excludeModulePaths` option was deprecated in favour of `indexing.ignorePaths`. `excludeModulePaths` is now ignored ([#1003](https://github.com/hashicorp/terraform-ls/pull/1003))
 - settings: `ignoreDirectoryNames` option was deprecated in favour of [`indexing.ignoreDirectoryNames`](https://github.com/hashicorp/terraform-ls/blob/v0.29.0/docs/SETTINGS.md#ignoredirectorynames-string). `ignoreDirectoryNames` is now ignored ([#1003](https://github.com/hashicorp/terraform-ls/pull/1003), [#1010](https://github.com/hashicorp/terraform-ls/pull/1010))
 - settings: `terraformExecPath` option was deprecated in favour of [`terraform.path`](https://github.com/hashicorp/terraform-ls/blob/v0.29.0/docs/SETTINGS.md#path-string). Old option is now ignored. ([#1011](https://github.com/hashicorp/terraform-ls/pull/1011))
 - settings: `terraformExecTimeout` option was deprecated in favour of [`terraform.timeout`](https://github.com/hashicorp/terraform-ls/blob/v0.29.0/docs/SETTINGS.md#timeout-string). Old option is now ignored. ([#1011](https://github.com/hashicorp/terraform-ls/pull/1011))
 - settings: `terraformLogFilePath` option was deprecated in favour of [`terraform.logFilePath`](https://github.com/hashicorp/terraform-ls/blob/v0.29.0/docs/SETTINGS.md#logfilepath-string). Old option is now ignored. ([#1011](https://github.com/hashicorp/terraform-ls/pull/1011))
 - cmd/serve: Previously deprecated `-tf-exec*` CLI flags were removed (`-tf-exec`, `-tf-exec-timeout` and `-tf-log-file`) in favour of LSP-based [`terraform.*`](https://github.com/hashicorp/terraform-ls/blob/main/docs/SETTINGS.md#terraform-object-) configuration options ([#1012](https://github.com/hashicorp/terraform-ls/pull/1012))

ENHANCEMENTS:

 - Replace internal watcher (used for watching changes in installed plugins and modules) with LSP dynamic capability registration & `workspace/didChangeWatchedFiles`. This should leave to improved performance in most cases. ([#953](https://github.com/hashicorp/terraform-ls/pull/953))
 - Provide completion, hover and docs links for uninitialized Registry modules ([#924](https://github.com/hashicorp/terraform-ls/pull/924))
 - Provide basic IntelliSense (except for diagnostics) for hidden `*.tf` files ([#971](https://github.com/hashicorp/terraform-ls/pull/971))
 - deps: bump terraform-schema to introduce v1.1 `terraform` `cloud` block ([terraform-schema#117](https://github.com/hashicorp/terraform-schema/pull/117))
 - deps: bump terraform-schema to introduce v1.1 `moved` block ([terraform-schema#121](https://github.com/hashicorp/terraform-schema/pull/121))
 - deps: bump terraform-schema to introduce v1.2 `lifecycle` conditions ([terraform-schema#115](https://github.com/hashicorp/terraform-schema/pull/115))
 - deps: bump terraform-schema to introduce v1.2 `lifecycle` `replace_triggered_by` ([terraform-schema#123](https://github.com/hashicorp/terraform-schema/pull/123))
 - Use `module` declarations from parsed configuration as source of truth for `module.calls` ([#987](https://github.com/hashicorp/terraform-ls/pull/987))
 - walker: Index uninitialized modules ([#997](https://github.com/hashicorp/terraform-ls/pull/997))
 - Recognize inputs and outputs of uninitialized local modules ([#598](https://github.com/hashicorp/terraform-ls/issues/598))
 - Enable go to module output declaration from reference ([#1007](https://github.com/hashicorp/terraform-ls/issues/1007))
 - settings: New option [`indexing.ignorePaths`](https://github.com/hashicorp/terraform-ls/blob/v0.29.0/docs/SETTINGS.md#ignorepaths-string) was introduced ([#1003](https://github.com/hashicorp/terraform-ls/pull/1003), [#1010](https://github.com/hashicorp/terraform-ls/pull/1010))
 - Introduce `module.terraform` custom LSP command to expose Terraform requirements & version ([#1016](https://github.com/hashicorp/terraform-ls/pull/1016))
 - Avoid obtaining schema via Terraform CLI if the same version is already cached (based on plugin lock file) ([#1014](https://github.com/hashicorp/terraform-ls/pull/1014))
 - Avoid getting version via `terraform version` during background indexing and pick relevant IntelliSense data based on `required_version` constraint ([#1027](https://github.com/hashicorp/terraform-ls/pull/1027))
 - Provide 0.12 based IntelliSense for any <0.12 Terraform versions ([#1027](https://github.com/hashicorp/terraform-ls/pull/1027))
 - Complete module source and version attributes for local and registry modules ([#1024](https://github.com/hashicorp/terraform-ls/pull/1024))

BUG FIXES:

 - handlers/command: Return partially parsed metadata from `module.providers` ([#951](https://github.com/hashicorp/terraform-ls/pull/951))
 - fix: Avoid ignoring hidden `*.tfvars` files ([#968](https://github.com/hashicorp/terraform-ls/pull/968))
 - fix: Avoid crash on invalid URIs ([#969](https://github.com/hashicorp/terraform-ls/pull/969))
 - fix: Avoid crash on invalid provider name ([#1030](https://github.com/hashicorp/terraform-ls/pull/1030))

INTERNAL:

 - job: introduce explicit priority for jobs ([#977](https://github.com/hashicorp/terraform-ls/pull/977))
 - main: allow build version metadata to be set ([#945](https://github.com/hashicorp/terraform-ls/pull/945))
 - deps: switch to the new minimal `terraform-registry-address` API ([#949](https://github.com/hashicorp/terraform-ls/pull/949))
 - deps: bump LSP structs to match gopls 0.8.4 ([#947](https://github.com/hashicorp/terraform-ls/pull/947))
 - deps: bump github.com/hashicorp/terraform-exec from 0.16.1 to 0.17.0 ([#963](https://github.com/hashicorp/terraform-ls/pull/963))
 - deps: bump github.com/hashicorp/go-version from 1.5.0 to 1.6.0 ([#979](https://github.com/hashicorp/terraform-ls/pull/979))
 - indexer: refactor & improve/cleanup error handling ([#988](https://github.com/hashicorp/terraform-ls/pull/988))
 - indexer/walker: Avoid running jobs where not needed ([#1006](https://github.com/hashicorp/terraform-ls/pull/1006))

## 0.28.1 (9 June 2022)

Due to some release pipeline changes and multiple release attempts, `0.28.0` release was published with checksums mismatching the release artifacts.

This release is therefore equivalent to `v0.28.0`, but published with the correct checksums.

## 0.28.0 (7 June 2022)

ENHANCEMENTS:
 - Link to documentation from module source for registry modules ([#874](https://github.com/hashicorp/terraform-ls/pull/874))
 - Provide refresh mechanism for `module.providers` when providers change ([#902](https://github.com/hashicorp/terraform-ls/pull/902))
 - Provide refresh mechanism for `module.calls` when module calls change ([#909](https://github.com/hashicorp/terraform-ls/pull/909))
 - Add support for `workspace/didChangeWatchedFiles` notifications for `*.tf` & `*.tfvars` ([#790](https://github.com/hashicorp/terraform-ls/pull/790))
 - Improve performance by reducing amount of notifications sent for any single module changes ([#931](https://github.com/hashicorp/terraform-ls/pull/931))

BUG FIXES:
 - Ignore duplicate document versions in `textDocument/didChange` ([#940](https://github.com/hashicorp/terraform-ls/pull/940))

INTERNAL:
 - build(deps): bump github.com/mitchellh/cli from 1.1.2 to 1.1.3 ([#886](https://github.com/hashicorp/terraform-ls/pull/886))
 - Use `terraform-registry-address` for parsing module sources ([#891](https://github.com/hashicorp/terraform-ls/pull/891))
 - Add utm parameters to docs links in `module.*` commands ([#923](https://github.com/hashicorp/terraform-ls/pull/923))

## 0.27.0 (14 April 2022)

NOTES / BREAKING CHANGES:

 - langserver/handlers/command: Remove `rootmodules` command ([#846](https://github.com/hashicorp/terraform-ls/pull/846))
 - cmd: Remove `completion` CLI command ([#852](https://github.com/hashicorp/terraform-ls/pull/852))

ENHANCEMENTS:

 - Provide (opt-in) custom semantic tokens & modifiers ([#833](https://github.com/hashicorp/terraform-ls/pull/833))
 - Enable 'go to module source' for local modules (via [#849](https://github.com/hashicorp/terraform-ls/pull/849))
 - Enable opening a single Terraform file ([#843](https://github.com/hashicorp/terraform-ls/pull/843))

BUG FIXES:

 - Avoid hanging when workspace contains >50 folders ([#839](https://github.com/hashicorp/terraform-ls/pull/839))
 - Make loading of parent directory after lower level directories work ([#851](https://github.com/hashicorp/terraform-ls/pull/851))
 - Fix corrupted diffs in formatting responses ([#876](https://github.com/hashicorp/terraform-ls/pull/876))
 - Fix `module.calls` command for Registry modules installed by Terraform v1.1+ ([#872](https://github.com/hashicorp/terraform-ls/pull/872))

INTERNAL:

 - Add job scheduler benchmarks & document [expectations around performance](https://github.com/hashicorp/terraform-ls/blob/v0.27.0/docs/benchmarks.md) ([#840](https://github.com/hashicorp/terraform-ls/pull/840))

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
