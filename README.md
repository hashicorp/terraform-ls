# Terraform Language Server

Experimental version of Terraform Language Server.

Not all LSP or language features are available at the time of writing as this is an active project with the aim of delivering smaller, incremental updates over time.

## What is Terraform?

[Terraform](https://www.terraform.io) enables you to safely and predictably create, change, and improve infrastructure. It is an open source tool that codifies APIs into declarative configuration files that can be shared amongst team members, treated as code, edited, reviewed, and versioned.

## What is Language Server?

A server implementing the [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) (LSP), which in turn defines the protocol used between an editor or IDE and a language server that provides language features like auto complete, go to definition, find all references etc.

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
 - Add the following snippet to the LSP settings' `clients`:

```json
"terraform": {
  "command": ["terraform-ls", "serve"],
  "enabled": true,
  "languageId": "terraform",
  "scopes": ["source.terraform"],
  "syntaxes": ["Packages/Terraform/Terraform.sublime-syntax"]
}
```

## Troubleshooting

The language server produces detailed logs which are send to stderr by default.
Most IDEs provide a way of inspecting these logs when server is launched in the standard
stdin/stdout mode.

Logs can also be redirected into file using flags of the `serve` command, e.g.

```sh
$ terraform-ls serve -log-file=/tmp/terraform-ls-{{pid}}.log -tf-log-file=/tmp/tf-exec-{{lsPid}}-{{args}}.log
```

It is recommended to inspect these logs when reporting bugs.

### Log Rotation

Keep in mind that the language server itself does not have any log rotation facility,
but the destination path will be truncated before being logged into.

Static paths may produce large files over the lifetime of the server and
templated paths (as described below) may produce many log files over time.

### Log Path Templating

Log paths support template syntax. This allows sane separation of logs while accounting for:

 - multiple server instances
 - multiple clients
 - multiple Terraform executions which may happen in parallel

**`-log-file`** supports the following functions:

 - `timestamp` - current timestamp (formatted as [`Time.Unix()`](https://golang.org/pkg/time/#Time.Unix), i.e. the number of seconds elapsed since January 1, 1970 UTC)
 - `pid` - process ID of the language server
 - `ppid` - parent process ID (typically editor's or editor plugin's PID)

 **`-tf-log-file`** supports the following functions:

  - `timestamp` - current timestamp (formatted as [`Time.Unix()`](https://golang.org/pkg/time/#Time.Unix), i.e. the number of seconds elapsed since January 1, 1970 UTC)
  - `lsPid` - process ID of the language server
  - `lsPpid` - parent process ID of the language server (typically editor's or editor plugin's PID)
  - `args` - all arguments passed to `terraform` turned into a safe `-` separated string

The path is interpreted as [Go template](https://golang.org/pkg/text/template/), e.g. `/tmp/terraform-ls-{{timestamp}}.log`.

## Contributing/Development

### Troubleshooting

[PacketSender](https://packetsender.com) enables you to open a TCP socket with a server, when launched as such.
Approximate steps of debugging follow.

 - Install PacketSender (e.g. on MacOS via `brew cask install packet-sender`)
 - Launch LS in TCP mode: `terraform-ls serve -port=8080`
 - Send any requests via PacketSender
   - Set `Address` to `127.0.0.1`
   - Set `Port` to `8080`
   - Tick `Persistent TCP`
   - Hit the `Send` button (which opens the TCP connection)
   - Paste or type request in LSP format (see below) & hit `Send`

Examples of formatted requests follow.

```
Content-Length: 164\n\n{"jsonrpc":"2.0","params":{"textDocument":{"uri":"file:///var/path/to/file/main.tf"},"position":{"line":1,"character":0}},"method":"textDocument/completion","id":2}
```
```
Content-Length: 72\n\n{"jsonrpc":"2.0","params":{"id":2},"method":"$/cancelRequest","id":null}
```
```
Content-Length: 47\n\n{"jsonrpc":"2.0","method":"shutdown","id":null}
```

### Notes

 - Keep in mind that each TCP session receives an isolated context,
    so you cannot cancel requests you didn't start yourself

## Credits

The implementation was inspired by:

 - [`juliosueiras/terraform-lsp`](https://github.com/juliosueiras/terraform-lsp)
 - [Martin Atkins](https://github.com/apparentlymart) (particularly the virtual filesystem)
