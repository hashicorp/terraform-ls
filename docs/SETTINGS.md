# Settings

## Supported Options

The language server supports the following configuration options:

## `terraformLogFilePath` (`string`)

Path to a file for Terraform executions to be logged into (`TF_LOG_PATH`)
with support for variables (e.g. Timestamp, Pid, Ppid) via Go template
syntax `{{.VarName}}`

## `terraformExecTimeout` (`string`)

Overrides Terraform execution timeout in [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration)
compatible format (e.g. `30s`)

## `terraformExecPath` (`string`)

Path to the Terraform binary.

This is usually looked up automatically from `$PATH` and should not need to be
specified in majority of cases. Use this to override the automatic lookup.

## **DEPRECATED**: `rootModulePaths` (`[]string`)

This option is deprecated and ignored from v0.29+, it was used as an escape hatch
to force indexing of paths in cases where indexer wouldn't index them otherwise.
Indexer in 0.29.0 no longer limited to just initialized modules (folders with `.terraform`)
and instead indexes all directories with `*.tf` files in them.
Therefore this option is no longer relevant.

If you previously used it to force indexing of a folder outside of a workspace,
you can just add that folder to the workspace and it will be indexed as usual.

## **DEPRECATED**: `excludeModulePaths` (`[]string`)

Deprecated in favour of `ignorePaths`

## `indexing.ignorePaths` (`[]string`)

Paths to ignore when indexing the workspace on initialization. This can serve
as an escape hatch in large workspaces. Key side effect of ignoring a path
is that go-to-definition, go-to-references and generally most IntelliSense
related to local `module` blocks will **not** work until the target module code
is explicitly opened.

Relative paths are resolved relative to the root (workspace) path opened in the editor.

Path separators are converted automatically to the match separators
of the target platform (e.g. `\` on Windows, or `/` on Unix),
symlinks are followed, trailing slashes automatically removed,
and `~` is replaced with your home directory.

## `commandPrefix`

Some clients such as VS Code keep a global registry of commands published by language
servers, and the names must be unique, even between terraform-ls instances. Setting
this allows multiple servers to run side by side, albeit the client is now responsible
for routing commands to the correct server. Users should not need to worry about
this, the frontend client extension should manage it.

The prefix will be applied to the front of the command name, which already contains
a `terraform-ls` prefix.

`commandPrefix.terraform-ls.commandName`

Or if left empty

`terraform-ls.commandName`

This setting should be deprecated once the language server supports multiple workspaces,
as this arises in VS code because a server instance is started per VS Code workspace.

## **DEPRECATED**: `ignoreDirectoryNames` (`[]string`)

Deprecated in favour of `indexing.ignoreDirectoryNames`

## `indexing.ignoreDirectoryNames` (`[]string`)

This allows excluding directories from being indexed upon initialization by passing a list of directory names.

The following list of directories will always be ignored:

- `.git`
- `.idea`
- `.vscode`
- `terraform.tfstate.d`
- `.terragrunt-cache`

## `ignoreSingleFileWarning` (`bool`)

This setting controls whether terraform-ls sends a warning about opening up a single Terraform file instead of a Terraform folder. Setting this to `true` will prevent the message being sent. The default value is `false`.

## `experimentalFeatures` (object)

This object contains inner settings used to opt into experimental features not yet ready to be on by default.

### `validateOnSave` (`bool`)

Enabling this feature will run terraform validate within the folder of the file saved. This comes with some user experience caveats.
 - Validation is not run on file open, only once it's saved.
 - When editing a module file, validation is not run due to not knowing which "rootmodule" to run validation from (there could be multiple). This creates an awkward workflow where when saving a file in a rootmodule, a diagnostic is raised in a module file. Editing the module file will not clear the diagnostic for the reason mentioned above, it will only clear once a file is saved back in the original "rootmodule". We will continue to attempt improve this user experience.

### `prefillRequiredFields` (`bool`)

Enables advanced completion for `provider`, `resource`, and `data` blocks where any required fields for that block are pre-filled. All such attributes and blocks are sorted alphabetically to ensure consistent ordering.

When disabled (unset or set to `false`), completion only provides the label name.

For example, when completing the `aws_appmesh_route` resource the `mesh_name`, `name`, `virtual_router_name` attributes and the `spec` block will fill and prompt you for appropriate values.

## How to pass settings

The server expects static settings to be passed as part of LSP `initialize` call,
but how settings are requested from on the UI side depends on the client.

### Sublime Text

Use `initializationOptions` key under the `clients.terraform` section, e.g.

```json
{
	"clients": {
		"terraform": {
			"initializationOptions": {
				"rootModulePaths": ["/any/path"]
			},
		}
	}
}
```
or
```json
{
	"clients": {
		"terraform": {
			"initializationOptions": {
				"excludeModulePaths": ["/any/path"]
			},
		}
	}
}
```

### VS Code

Use `terraform-ls`, e.g.

```json
{
	"terraform-ls": {
		"rootModulePaths": ["/any/path"]
	}
}
```
or
```json
{
	"terraform-ls": {
		"excludeRootModules": ["/any/path"]
	}
}
