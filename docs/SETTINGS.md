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

## `rootModulePaths` (`[]string`)

This allows overriding automatic root module discovery by passing a static list
of absolute or relative paths to root modules (i.e. folders with `*.tf` files
which have been `terraform init`-ed). Conflicts with `ExcludeModulePaths` option.

Relative paths are resolved relative to the directory opened in the editor.

Path separators are converted automatically to the match separators
of the target platform (e.g. `\` on Windows, or `/` on Unix),
symlinks are followed, trailing slashes automatically removed,
and `~` is replaced with your home directory.

## `excludeModulePaths` (`[]string`)

This allows exclude root module path when automatic root module discovery by passing a static list
of absolute or relative paths to root modules (i.e. folders with `*.tf` files
which have been `terraform init`-ed). Conflicts with `rootModulePaths` option.

Relative paths are resolved relative to the directory opened in the editor.

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

## `experimentalFeatures`

This setting contains inner settings used to opt into experimental features not yet ready to be on by default.

### `experimentalFeatures.validateOnSave`

Enabling this feature will run terraform validate within the folder of the file saved. This comes with some user experience caveats.
 - Validation is not run on file open, only once it's saved.
 - When editing a module file, validation is not run due to not knowing which "rootmodule" to run validation from (there could be multiple). This creates an awkward workflow where when saving a file in a rootmodule, a diagnostic is raised in a module file. Editing the module file will not clear the diagnostic for the reason mentioned above, it will only clear once a file is saved back in the original "rootmodule". We will continue to attempt improve this user experience.

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
