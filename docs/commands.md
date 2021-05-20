# Commands

The server exposes the following executable commands via LSP to clients.
Typically these commands are not invokable by end-users automatically.
Instead this serves as a documentation for client maintainers,
and clients may expose these e.g. via command palette where appropriate.

Every care is taken to avoid breaking changes, but these interfaces
should not be considered stable yet and may change.

Either way clients should always follow LSP spec in the sense
that they check whether a command is actually supported or not
(via `ServerCapabilities.executeCommandProvider.commands`).

## Command Prefix

All commands use `terraform-ls.` prefix to avoid any conflicts
with commands registered by any other language servers user
may be using at the same time.

Some clients may also choose to generate additional prefix
where e.g. the language server runs in multiple instances
and registering the same commands would lead to conflicts.

This can be passed as part of `initializationOptions`,
as documented in [Settings](./SETTINGS.md#commandprefix).

## Arguments

All commands accept arguments as string arrays with `=` used
as a separator between key and value. i.e.

```json
{
	"command": "command-name",
	"arguments": [ "key=value" ]
}
```

## Supported Commands

### `terraform.init`

Runs [`terraform init`](https://www.terraform.io/docs/cli/commands/init.html) using available `terraform` installation from `$PATH`.

**Arguments:**

 - `uri` - URI of the directory in which to run `terraform init`

**Outputs:**

Error is returned e.g. when `terraform` is not installed, or when execution fails,
but no output is returned if `init` successfully finishes.

### `terraform.validate`

Runs [`terraform validate`](https://www.terraform.io/docs/cli/commands/validate.html) using available `terraform` installation from `$PATH`.

Any violations are published back the the client via [`textDocument/publishDiagnostics` notification](https://microsoft.github.io/language-server-protocol/specifications/specification-current/#textDocument_publishDiagnostics).

Diagnostics are not persisted and any document change will cause them to be lost.

**Arguments:**

 - `uri` - URI of the directory in which to run `terraform validate`

**Outputs:**

Error is returned e.g. when `terraform` is not installed, or when execution fails,
but no output is returned if `validate` successfully finishes.

### `module.callers`

In Terraform module hierarchy "callers" are modules which _call_ another module
via `module "..." {` blocks.

Language server will attempt to discover any module hierarchy within the workspace
and this command can be used to obtain the data about such hierarchy, which
can be used to hint the user e.g. where to run `init` or `validate` from.

**Arguments:**

 - `uri` - URI of the directory of the module in question, e.g. `file:///path/to/network`

**Outputs:**

 - `v` - describes version of the format; Will be used in the future to communicate format changes.
 - `callers` - array of any modules found in the workspace which call the module in question
   - `uri` - URI of the directory (absolute path)

```json
{
	"v": 0,
	"callers": [
		{
			"uri": "file:///path/to/dev",
		},
		{
			"uri": "file:///path/to/prod",
		}
	]
}
```

### `rootmodules` (DEPRECATED, use `module.callers` instead)
