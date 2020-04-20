# Usage of Terraform Language Server

This guide assumes you have installed the server by following instructions
in the [README.md](../README.md) if that is applicable to your client
(i.e. if the client doesn't download the server itself).

Instructions for popular IDEs are below and pull requests
for updates or addition of more IDEs are welcomed.

## Sublime Text 2

 - Install the [LSP package](https://github.com/sublimelsp/LSP#installation)
 - Add the following snippet to your _User_ `LSP.sublime-settings` (editable via `Preferences → Package Settings → LSP → Settings` or via the command pallete → `Preferences: LSP Settings`)

```json
{
	"clients": {
		"terraform": {
			"command": ["terraform-ls", "serve"],
			"enabled": true,
			"languageId": "terraform",
			"scopes": ["source.terraform"],
			"syntaxes": ["Packages/Terraform/Terraform.sublime-syntax"]
		}
	}
}
```

## NeoVim

 - Install the [coc.nvim plugin](https://github.com/neoclide/coc.nvim)
 - Add the following snippet to the `coc-setting.json` file (editable via `:CocConfig` in NeoVim)

```json
{
	"languageserver": {
		"terraform": {
			"command": "terraform-ls",
			"args": ["serve"],
			"filetypes": [
				"terraform",
				"tf"
			],
			"initializationOptions": {},
			"settings": {}
		}
	}
}
```

Make sure to read through the [example vim configuration](https://github.com/neoclide/coc.nvim#example-vim-configuration) of the plugin, especially key remapping, which is required for completion to work correctly:

```vim
" Use <c-space> to trigger completion.
inoremap <silent><expr> <c-space> coc#refresh()
```
