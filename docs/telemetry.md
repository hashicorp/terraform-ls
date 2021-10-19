# Telemetry

The language server is capable of sending telemetry using the native LSP `telemetry/event` method.
Telemetry is off by default and can enabled by passing a supported request format version
as an experimental client capability.

```json
{
    "capabilities": {
        "experimental": {
            "telemetryVersion": 1
        }
    }
}
```

Clients then implement opt-in or opt-out in the UI and should reflect the user's choice.

## Privacy

Sensitive data, such as filesystem paths or addresses of providers sourced from outside the Terraform Registry
are anonymized. Random UUID is generated in memory and tracked instead of a path or a private provider address.

Mapping of such UUIDs is not persisted anywhere other than in memory during process lifetime.

## Request Format

The only supported version is currently `1`. Version negotiation allows the server
to introduce breaking changes to the format and have clients adopt gradually.

### `v1`

[`telemetry/event`](https://microsoft.github.io/language-server-protocol/specifications/specification-3-16/#telemetry_event) structure

```json
{
	"v": 1,
	"name": "eventName",
	"properties": {}
}
```

`properties` may contain **any (valid) JSON types**
including arrays and arbitrarily nested objects. It is client's
reponsibility to serialize these properties when and if necessary.

Example events:

```json
{
    "v": 1,
    "name": "initialize",
    "properties": {
        "experimentalCapabilities.referenceCountCodeLens": true,
        "lsVersion": "0.23.0",
        "options.commandPrefix": true,
        "options.excludeModulePaths": false,
        "options.experimentalFeatures.prefillRequiredFields": false,
        "options.experimentalFeatures.validateOnSave": false,
        "options.rootModulePaths": false,
        "options.terraformExecPath": false,
        "options.terraformExecTimeout": "",
        "options.terraformLogFilePath": false,
        "root_uri": "dir"
    }
}
```
```json
{
    "v": 1,
    "name": "moduleData",
    "properties": {
        "backend": "remote",
        "backend.remote.hostname": "app.terraform.io",
        "installedProviders": {
            "registry.terraform.io/hashicorp/aws": "3.57.0",
            "registry.terraform.io/hashicorp/null": "3.1.0"
        },
        "moduleId": "8aa5a4dc-4780-2d90-b8fb-57de8288fb32",
        "providerRequirements": {
            "registry.terraform.io/hashicorp/aws": "",
            "registry.terraform.io/hashicorp/null": "~> 3.1"
        },
        "tfRequirements": "~> 1.0",
        "tfVersion": "1.0.7"
    }
}
```
