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

### `module.calls`

List of modules called by the module under the given URI.

Empty array may be returned when e.g.
  - the URI doesn't represent a module
  - the configuration is invalid
  - there are no module calls

The data is sourced from the declared modules inside the files of the module.

**Arguments:**

 - `uri` - URI of the directory of the module in question, e.g. `file:///path/to/network`

**Outputs:**

 - `v` - describes version of the format; Will be used in the future to communicate format changes.
 - `module_calls` - array of modules which are called from the module in question
   - `name` - the reference name of this particular module (i.e. `network` from `module "network" { ...`)
   - `source_addr` - the source address given for this module call (e.g. `terraform-aws-modules/eks/aws`)
   - `version` - version constraint of the module call; applicable to modules hosted by the Terraform Registry (e.g. `~> 1.0`
   - `source_type` - source of the Terraform module, e.g. `github` or `tfregistry`
   - `docs_link` - a link to the module documentation; if available
   - `dependent_modules` - **DEPRECATED** (always empty in `v0.29+`) - array of dependent modules with the same structure as `module_calls`

```json
{
  "v": 0,
  "module_calls": [
    {
      "name": "child",
      "source_addr": "./child",
      "source_type": "local",
      "dependent_modules": []
    },
    {
      "name": "vpc",
      "source_addr": "terraform-aws-modules/vpc/aws",
      "version": "3.11.0",
      "source_type": "tfregistry",
      "docs_link": "https://registry.terraform.io/modules/terraform-aws-modules/vpc/aws/3.11.0",
      "dependent_modules": []
    }
  ]
}
```

### `module.providers`

Provides information about the providers of the current module, including requirements and
installed version.

**Arguments:**

 - `uri` - URI of the directory of the module in question, e.g. `file:///path/to/network`

**Outputs:**

 - `v` - describes version of the format; Will be used in the future to communicate format changes.
 - `provider_requirements` - map of provider FQN string to requirements object
   - `display_name` - a human-readable name of the provider (e.g. `hashicorp/aws`)
   - `version_constraint` - a comma-separated list of version constraints (e.g. `>= 1.0, < 1.2`)
   - `docs_link` - a link to the provider documentation; if available
 - `installed_providers` - map where _key_ is the provider FQN and _value_ is the installed version of the provider; can be empty if none are installed

```json
{
  "v": 0,
  "provider_requirements": {
    "registry.terraform.io/hashicorp/aws": {
      "display_name": "hashicorp/aws",
      "version_constraint": "~> 3.64.0",
      "docs_link": "https://registry.terraform.io/providers/hashicorp/aws/latest"
    }
  },
  "installed_providers": {
    "registry.terraform.io/hashicorp/aws": "3.64.2"
  }
}
```
