# Terraform LS

Experimental prototype of Terraform language server.

## Disclaimer

If you found this repo via GitHub, there's likely nothing to see here for you, at least not yet.

This project is likely to change, move or disappear without prior notice.
Expect no support or stability at this point.

The implementation is intentionally minimal just to initially see what's possible
without having to import Terraform and tightly couple the LS with it.

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
