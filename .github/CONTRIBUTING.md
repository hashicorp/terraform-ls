# Contributing to Terraform Language Server

## Reporting Feedback

Terraform language server is an open source project and we appreciate
contributions of various kinds, including bug reports and fixes,
enhancement proposals, documentation updates, and user experience feedback.

To record a bug report, enhancement proposal, or give any other product
feedback, please [open a GitHub issue](https://github.com/hashicorp/terraform-ls/issues/new/choose)
using the most appropriate issue template. Please do fill in all of the
information the issue templates request, because we've seen from experience that
this will maximize the chance that we'll be able to act on your feedback.

**All communication on GitHub, the community forum, and other HashiCorp-provided
communication channels is subject to
[the HashiCorp community guidelines](https://www.hashicorp.com/community-guidelines).**

## Scope

This repository contains the source code only for Terraform language server,
which in turn relies on other projects that have their own repositories.

[Terraform CLI/core has its own repository.](https://github.com/hashicorp/terraform)

The HashiCorp-maintained Terraform providers are open source but are **not**
in this repository; instead, they are each in their own repository in
[the `terraform-providers` organization](https://github.com/terraform-providers)
on GitHub.

This repository also does **not** include the source code for some other parts of
the Terraform product including Terraform Cloud, Terraform Enterprise, and the
Terraform Registry. Those components are not open source, though if you have
feedback about them (including bug reports) please do feel free to
[open a GitHub issue in the core repository](https://github.com/hashicorp/terraform/issues/new/choose).

## Development

If you wish to work on the source code, you'll first need to install
 the [Go](https://golang.org/) compiler and the version control system
[Git](https://git-scm.com/).

Refer to the file [`.go-version`](.go-version) to see which version of Go
the language server is currently built with. Other versions will often work,
but if you run into any build or testing problems please try with the specific
Go version indicated. You can optionally simplify the installation of multiple
specific versions of Go on your system by installing
[`goenv`](https://github.com/syndbg/goenv), which reads `.go-version` and
automatically selects the correct Go version.

Use Git to clone this repository into a location of your choice. Dependencies
are tracked via [Go Modules](https://blog.golang.org/using-go-modules),
and so you should _not_ clone it inside your `GOPATH`.

Switch into the root directory of the cloned repository and build
the language server

```
cd terraform-ls
go install
```

Once the compilation process succeeds, you can find a `terraform-ls` executable in
the Go executable directory. If you haven't overridden it with the `GOBIN`
environment variable, the executable directory is the `bin` directory inside
the directory returned by the following command:

```
go env GOPATH
```

If you are planning to make changes to the source code, you should run the
unit test suite before you start to make sure everything is initially passing:

```
go test ./...
```

As you make your changes, you can re-run the above command to ensure that the
tests are _still_ passing. If you are working only on a specific Go package,
you can speed up your testing cycle by testing only that single package, or
packages under a particular package prefix:

```
go test ./internal/terraform/exec/...
go test ./langserver
```

## External Dependencies

Terraform uses [Go Modules]((https://blog.golang.org/using-go-modules))
for dependency management.

If you need to add a new dependency to Terraform or update the selected version
for an existing one, use `go get` from the root of the Terraform repository
as follows:

```
go get github.com/hashicorp/hcl/v2@2.0.0
```

This command will download the requested version (2.0.0 in the above example)
and record that version selection in the `go.mod` file. It will also record
checksums for the module in the `go.sum`.

To complete the dependency change, clean up any redundancy in the module
metadata files by running the following command:

```
go mod tidy
```

Because dependency changes affect a shared, top-level file, they are more likely
than some other change types to become conflicted with other proposed changes
during the code review process. For that reason, and to make dependency changes
more visible in the change history, we prefer to record dependency changes as
separate commits that include only the results of the above commands and the
minimal set of changes to the language server's own code for compatibility
with the new version:

```
git add go.mod go.sum
git commit -m "deps: go get github.com/hashicorp/hcl/v2@2.0.0"
```

You can then make use of the new or updated dependency in new code added in
subsequent commits.

### Licensing Policy

Our dependency licensing policy excludes proprietary licenses and "copyleft"-style
licenses. We accept the common Mozilla Public License v2, MIT License,
and BSD licenses. We will consider other open source licenses
in similar spirit to those three, but if you plan to include such a dependency
in a contribution we'd recommend opening a GitHub issue first to discuss what
you intend to implement and what dependencies it will require so that the
maintainer team can review the relevant licenses to for whether
they meet our licensing needs.

## Debugging

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

Keep in mind that each TCP session receives an isolated context,
so you cannot cancel requests you didn't start yourself

## Proposing a Change

If you'd like to contribute a code change, we'd love to review a GitHub pull request.

In order to be respectful of the time of community contributors, we prefer to
discuss potential changes in GitHub issues prior to implementation. That will
allow us to give design feedback up front and set expectations about the scope
of the change, and, for larger changes, how best to approach the work such that
the maintainer team can review it and merge it along with other concurrent work.

If the bug you wish to fix or enhancement you wish to implement isn't already
covered by a GitHub issue that contains feedback from the maintainer team,
please do start a discussion (either in
[a new GitHub issue](https://github.com/hashicorp/terraform-ls/issues/new/choose)
or an existing one, as appropriate) before you invest significant development
time. If you mention your intent to implement the change described in your
issue, the maintainer team can prioritize including implementation-related
feedback in the subsequent discussion.

Most changes will involve updates to the test suite, and changes to the
documentation. The maintainer team can advise on different testing strategies
for specific scenarios, and may ask you to revise the specific phrasing of
your proposed documentation prose to match better with the standard "voice" of
Terraform's documentation.

This repository is primarily maintained by a small team at HashiCorp along with
their other responsibilities, so unfortunately we cannot always respond
promptly to pull requests, particularly if they do not relate to an existing
GitHub issue where the maintainer team has already participated. We _are_
grateful for all contributions however, and will give feedback on pull requests
as soon as we're able to.
