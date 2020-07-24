// Package filesystem implements a virtual filesystem which reflects
// the needs of both the language server and the HCL parser.
//
// - creates in-memory files based on data received from the language client
// - allows updating in-memory files via diffs received from the language client
// - maintains file metadata (e.g. version, or whether it's open by the client)
package filesystem
