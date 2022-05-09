# Language Server Implementation Notes

## How to add an Experimental Capability

Add a new entry to the ExperimentalServerCapabilities struct in `internal/protocol/expertimental.go:6`:

```go
type ExperimentalServerCapabilities struct {
  ReferenceCountCodeLens bool `json:"referenceCountCodeLens"`
  RefreshModuleProviders bool `json:"refreshModuleProviders"`
  RefreshModuleCalls     bool `json:"refreshModuleCalls"`
}
```

> Note the casing in the mapstructure field compared to the field name.

Add a new method to retrieve the client side command id in `internal/protocol/expertimental.go`:

```go
func (cc ExpClientCapabilities) NewItemHereCommandId() (string, bool) {
  if cc == nil {
    return "", false
  }

  cmdId, ok := cc["newItemHereCommandId"].(string)
  return cmdId, ok
}
```

> Note that the command ID matches the experimental capabilities struct in expertimental.go`

Add a new stanza to `internal/langServer/handlers/initialize.go:63` to pull the command ID and register the capability:

```go
if _, ok := expClientCaps.NewItemHereCommandId(); ok {
  expServerCaps.NewItemHere = true
  properties["experimentalCapabilities.newItemHere"] = true
}
```

> Note the casing in the proprties hash.

Finally, register the command handler in `internal/langServer/handlers/service.go:454`:

```go
if commandId, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).NewItemHereCommandId(); ok {
  // do something with commandId here
}
```
