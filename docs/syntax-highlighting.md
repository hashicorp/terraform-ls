# Syntax Highlighting

Highlighting syntax is one of the key features expected of any editor. Editors typically have a few different solutions to choose from. Below is our view on how we expect editors to highlight Terraform code while using this language server.

## Static Grammar

Highlighting Terraform language syntax via static grammar (such as TextMate) _accurately_ may be challenging but brings more immediate value to the end user, since starting language server may take time. Also not all language clients may implement semantic token based highlighting.

HashiCorp maintains a set of grammars in https://github.com/hashicorp/syntax and we encourage you to use the available Terraform grammar as the *primary* way of highlighting the Terraform language.

## Semantic Tokens

[LSP (Language Server Protocol) 3.16](https://microsoft.github.io/language-server-protocol/specifications/specification-3-16/) introduced language server-driven highlighting. This language server is better equipped to provide more contextual and accurate highlighting as it can parse the whole AST, unlike a TextMate grammar operating on a regex-basis.

LSP 3.17 does support use cases where semantic highlighting is the only way to highlight a file (through [`augmentsSyntaxTokens` client capability](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#semanticTokensClientCapabilities)). However in the context of the Terraform language we recommend semantic highlighting to be used as in *addition* to a static grammar - i.e. this server does _not_ support `augmentsSyntaxTokens: false` mode and is not expected to be used in isolation to highlight configuration.

There are two main use cases we're targeting with semantic tokens.

### Improving Accuracy

Regex-based grammars (like TextMate) operate on line-basis, which makes it difficult to accurately highlight certain parts of the syntax, for example nested blocks occurring in the Terraform language (as below).

```hcl
terraform {
	required_providers {

	}
}
```

Language server can use the AST and other important context (such as Terraform version or provider schema) to fully understand the whole configuration and provide more accurate highlighting.

### Custom Theme Support

Many _default_ IDE themes are intended as general-purpose themes, highlighting token types, modifiers and scopes mappable to most languages. We recognize that theme authors would benefit from token types & modifiers which more accurately reflect the Terraform language.

LSP spec doesn't _explicitly_ encourage defining custom token types or modifiers, however the default token types and modifiers which are part of the spec are not well suited to express all the different constructs of a DSL (Domain Specific Language), such as Terraform language. With that in mind we use the LSP client/server capability negotiation mechanism to provide the following custom token types & modifiers with fallback to the predefined ones.

#### Token Types

Primary token types are preferred if deemed supported by client per `SemanticTokensClientCapabilities.TokenTypes`, fallbacks are also only reported if client claim support (using the same capability).

Fallback types are chosen based on meaningful semantic mapping and default themes in VSCode.

| Primary | Fallback |
| ------- | -------- |
| `hcl-blockType` | `type` |
| `hcl-blockLabel` | `enumMember` |
| `hcl-attrName` | `property` |
| `hcl-bool` | `keyword` |
| `hcl-number` | `number` |
| `hcl-string` | `string` |
| `hcl-objectKey` | `parameter` |
| `hcl-mapKey` | `parameter` |
| `hcl-keyword` | `variable` |
| `hcl-traversalStep` | `variable` |
| `hcl-typeComplex` | `function` |
| `hcl-typePrimitive` | `keyword` |
| `hcl-functionName` | `function` |

#### Token Modifiers

Modifiers which do not have fallback are not reported at all if not received within `SemanticTokensClientCapabilities.TokenModifiers` (just like fallback modifier that isn't supported).

| Primary | Fallback |
| ------- | -------- |
| `hcl-dependent` | `defaultLibrary` |
| `terraform-data` |  |
| `terraform-locals` |  |
| `terraform-module` |  |
| `terraform-output` |  |
| `terraform-provider` |  |
| `terraform-resource` |  |
| `terraform-provisioner` |  |
| `terraform-connection` |  |
| `terraform-variable` |  |
| `terraform-terraform` |  |
| `terraform-backend` |  |
| `terraform-name` |  |
| `terraform-type` |  |
| `terraform-requiredProviders` |  |
