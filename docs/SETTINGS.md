# Settings

## Supported Options

The language server supports the following configuration options:

## `rootModulePaths` (`[]string`)

This allows overriding automatic root module discovery by passing a static list
of absolute or relative paths to root modules (i.e. folders with `*.tf` files
which have been `terraform init`-ed).

Relative paths are resolved relative to the directory opened in the editor.

Path separators are converted automatically to the match separators
of the target platform (e.g. `\` on Windows, or `/` on Unix),
symlinks are followed, trailing slashes automatically removed,
and `~` is replaced with your home directory.

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

### VS Code

Use `terraform-ls`, e.g.

```json
{
	"terraform-ls": {
		"rootModulePaths": ["/any/path"]
	}
}
```
