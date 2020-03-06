# Terraform LS

Experimental prototype of Terraform language server.

## Disclaimer

This is not an officially supported HashiCorp product.

## How to try it out

```
go install .
```

This should produce a binary called `terraform-ls` in `$GOBIN/terraform-ls`.

Putting `$GOBIN` in your `$PATH` may save you from having to specify
absolute path to the binary.

### Visual Studio Code

Try https://github.com/aeschright/tf-vscode-demo/pull/1 - instructions are in that PR.

### Sublime Text 2

 - Install the [LSP package](https://github.com/sublimelsp/LSP#installation)
 - Add the following snippet to the LSP settings' clients:

```json
"terraform": {
  "command": ["terraform-ls", "serve"],
  "enabled": true,
  "languageId": "terraform",
  "scopes": ["source.terraform"],
  "syntaxes":  ["Packages/Terraform/Terraform.sublime-syntax"]
}
```

## Credits

The implementation was inspired by:

 - [`juliosueiras/terraform-lsp`](https://github.com/juliosueiras/terraform-lsp)
 - [Martin Atkins](https://github.com/apparentlymart) (particularly the virtual filesystem)
