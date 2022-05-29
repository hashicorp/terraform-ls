# Usage of Terraform Language Server

This guide assumes you have installed the server by following instructions
in the [README.md](../README.md) if that is applicable to your client
(i.e. if the client doesn't download the server itself).

Instructions for popular IDEs are below and pull requests
for updates or addition of more IDEs are welcomed.

See also [settings](./SETTINGS.md) to understand
how you may configure the settings.

## Emacs

If you are using `use-package`, you can use something like this:

``` emacs-lisp
(use-package lsp-mode
  :ensure t
  :hook ((terraform-mode . lsp-deferred)))
```

The above code installs `lsp-mode` and also enables it for the
terraform-mode.

The terraform client for lsp-mode supports various features like
semantic tokens, code lens for references etc. There is more detailed
documentation [here](https://emacs-lsp.github.io/lsp-mode/page/lsp-terraform-ls/).

## IntelliJ IDE

 - Install [LSP Support plugin](https://plugins.jetbrains.com/plugin/10209-lsp-support)
 - Open Settings
 - Go to `Languages & Frameworks → Language Server Protocol → Server Definitions`
   - Pick `Executable`
   - set `Extension` to `tf`
   - set `Path` to `terraform-ls`
   - set `Args` to `serve`
 - Confirm by clicking `Apply`

Please note that the [Terraform plugin](https://plugins.jetbrains.com/plugin/7808-hashicorp-terraform--hcl-language-support)
provides overlapping functionality (and more features at the time of writing).
As a result having both enabled at the same time may result in suboptimal UX,
such as duplicate completion candidates.

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

## Vim / NeoVim
### coc.nvim

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

### vim-lsp

 - Install the following plugins:
   * [async.vim plugin](https://github.com/prabirshrestha/async.vim)
   * [vim-lsp plugin](https://github.com/prabirshrestha/vim-lsp)
   * [asyncomplete.vim plugin](https://github.com/prabirshrestha/asyncomplete.vim)
   * [asyncomplete-lsp.vim plugin](https://github.com/prabirshrestha/asyncomplete-lsp.vim)
 - Add the following to your `.vimrc`:

```vim
if executable('terraform-ls')
    au User lsp_setup call lsp#register_server({
        \ 'name': 'terraform-ls',
        \ 'cmd': {server_info->['terraform-ls', 'serve']},
        \ 'whitelist': ['terraform'],
        \ })
endif
```

### LanguageClient-neovim

 - Install the [LanguageClient-neovim plugin](https://github.com/autozimu/LanguageClient-neovim)
 - Add the following to your `.vimrc`:

```vim
let g:LanguageClient_serverCommands = {
    \ 'terraform': ['terraform-ls', 'serve'],
    \ }
```

### Neovim v0.5.0+

 - Install the [nvim-lspconfig plugin](https://github.com/neovim/nvim-lspconfig)
 - Add the following to your `.vimrc`:

```vim
lua <<EOF
  require'lspconfig'.terraformls.setup{} 
EOF
autocmd BufWritePre *.tf lua vim.lsp.buf.formatting_sync()
```

Make sure to read through to [server_configurations.md#terraformls](https://github.com/neovim/nvim-lspconfig/blob/master/doc/server_configurations.md#terraformls) if you need more detailed settings.

## VS Code

 - Install [Terraform VS Code Extension](https://marketplace.visualstudio.com/items?itemName=hashicorp.terraform) `>=2.0.0`
 - Latest version of the language server should be installed automatically on VS Code launch
 - In case you are upgrading from v1 you may need to reset the settings to reflect v2 default, as shown below

```json
"terraform.languageServer": {
    "args": [
        "serve",
    ],
},
```

## BBEdit

*BBEdit 14 [added support](https://www.barebones.com/support/bbedit/lsp-notes.html) for the Language Server Protocol so you'll need to upgrade to version 14 to use; this won't work for older versions of BBEdit*.

- Open Preferences > Languages
- In *Language-specific settings* section, add an entry for Terraform
- In the Server tab, Set *Command* to `terraform-ls` and *Arguments* to `serve`
- Once you've correctly installed `terraform-ls` and configured BBEdit, the status indicator on this settings panel will flip to green
- If you'd like to pass any [settings](./SETTINGS.md) to the server you can do so via the *Arguments* field.
