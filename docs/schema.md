# Why and how does the language server pull provider schemas?

The Terraform Language Server needs to know about the names and structure of resources, data sources, provider blocks and provider-defined functions so that they can be suggested in completions or have their syntax validated. It retrieves this information from these sources:

- Terraform built-in schema
- Bundled provider schemas at compile time
- Obtained from locally installed providers

The language server comes with a built-in Terraform schema that handles resolving core Terraform constructs like resource, data, provider, backends, etc. We keep up with the release cadence of Terraform, from `0.12` to the latest, and detect which schema to use based on the `required_version` constraint in the declared `terraform` block and the locally installed Terraform version.

The language server comes with "bundled" schemas that are stored in terraform-ls at compile time (officially released HashiCorp binaries). Since they are stored in the binary, terraform-ls does not need providers to be installed locally. Only the schemas of the latest version for the most common Terraform providers (all official and partner providers) are included because it makes the binary larger in size than it would otherwise be. While this has the advantage that most providers work out of the box, it has the disadvantage that only one specific version is bundled. If a user uses an older (or newer) version of the provider, they are likely to run into schema validation errors.

The language server can also use locally installed providers in the `.terraform/providers` directory to get schema information. This is usually available after a user has run `terraform init`, which installs the provider binaries from the Terraform Registry. The language server will then obtain the schemas for all installed providers by executing the `terraform providers schema -json` command. This will result in the most accurate schema representation since the provider version is an exact match.

## Multi-Root Workspaces

Provider schema selection is done on a best effort basis. We always try to pick the best matching version for the given provider constraints. For complex multi-root workspaces, this is difficult to get right, especially when modules don’t have a direct link to a root module.

In this example, we currently don’t support picking the correct provider version for a module and may end up using the wrong provider version:

```
environments
├── production
│   └── .terraform  // has hashicorp/aws version 5.68.0 installed
└── staging
    └── .terraform  // has hashicorp/aws version 5.72.1 installed
modules
├── a
├── b
└── c               // <- which version to use here?
```

## Unexpected Attribute Errors

The language server has a feature called “Enhanced validation”, where it compares the actual content of the configuration with the internal schemas. We treat the internal schema as the source of truth, so whenever we detect an extraneous or missing attribute or block, we raise an error. But if our internal schema versions don't match the exact version of a provider in use, this can lead to false negatives.

# What about module schemas?

Similar to provider schemas, the language server also handles module schemas. Module schemas mainly contain the inputs and outputs of a module. They power completion inside a module block and references to module outputs. It retrieves this information from these sources:

- Reading local module files from disk
- Dynamically retrieving module schemas from the Terraform Registry
- Obtaining from locally installed remote modules

For local modules that reference a local path in their `source` attribute, the language server will parse that module directory and extract all defined variables and outputs and their types.

If a module source specifies a module that’s available in the **public** Terraform Registry, the language server will use the Registry API to fetch the module’s inputs and outputs.

For all module sources (Public Registry, Private Registry, Git, GitHub, …) installed locally via `terraform init`, the language server can parse the module manifest (`.terraform/modules/modules.json`) and identify the installation location. It then parses the content in a similar way to local modules.
