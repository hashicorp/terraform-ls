// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

const reindexVariables = `variable "first" {
}

variable "second" {
}
`

// The same two variables with three lines added above them, so each
// declaration starts three lines further down.
const reindexVariablesMoved = `# first comment
# second comment
# third comment
variable "first" {
}

variable "second" {
}
`

const reindexOutputFirst = `output "foo" {
  value = var.first
}
`

const reindexOutputSecond = `output "foo" {
  value = var.second
}
`

// reindexServer starts a server on tmpDir, which must already hold its files,
// and returns functions to send requests, wait for indexing and check where go
// to definition on line 1, character 14 of main.tf leads.
func reindexServer(t *testing.T, tmpDir document.DirHandle) (call func(method, params string), wait func(), expectDefinition func(id int, startLine, endLine int), stop func()) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop = ls.Start(t)

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
			"capabilities": {},
			"rootUri": %q,
			"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	call = func(method, params string) {
		ls.Call(t, &langserver.CallRequest{Method: method, ReqParams: params})
	}
	wait = func() {
		waitForAllJobs(t, ss)
	}
	expectDefinition = func(id int, startLine, endLine int) {
		ls.CallAndExpectResponse(t, &langserver.CallRequest{
			Method: "textDocument/definition",
			ReqParams: fmt.Sprintf(`{
				"textDocument": {"uri": "%s/main.tf"},
				"position": {"line": 1, "character": 14}
			}`, tmpDir.URI)}, fmt.Sprintf(`{
				"jsonrpc": "2.0",
				"id": %d,
				"result": [{
					"uri": "%s/variables.tf",
					"range": {
						"start": {"line": %d, "character": 0},
						"end": {"line": %d, "character": 1}
					}
				}]
			}`, id, tmpDir.URI, startLine, endLine))
	}
	return call, wait, expectDefinition, stop
}

func writeReindexFile(t *testing.T, tmpDir document.DirHandle, name, content string) {
	err := os.WriteFile(filepath.Join(tmpDir.Path(), name), []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

func openParams(tmpDir document.DirHandle, name, content string) string {
	return fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": %q,
			"uri": "%s/%s"
		}
	}`, content, tmpDir.URI, name)
}

func closeParams(tmpDir document.DirHandle, name string) string {
	return fmt.Sprintf(`{"textDocument": {"uri": "%s/%s"}}`, tmpDir.URI, name)
}

func changedOnDiskParams(tmpDir document.DirHandle, name string) string {
	return fmt.Sprintf(`{"changes": [{"uri": "%s/%s", "type": 2}]}`, tmpDir.URI, name)
}

// The file changes on disk while it is open, and the client reloads it by
// closing it and opening it again with the new content.
func TestDidOpen_reopenAfterReloadUsesNewContent(t *testing.T) {
	tmpDir := TempDir(t)
	writeReindexFile(t, tmpDir, "variables.tf", reindexVariables)
	writeReindexFile(t, tmpDir, "main.tf", reindexOutputFirst)

	call, wait, expectDefinition, stop := reindexServer(t, tmpDir)
	defer stop()

	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputFirst)) // id 2
	wait()

	writeReindexFile(t, tmpDir, "main.tf", reindexOutputSecond)
	call("textDocument/didClose", closeParams(tmpDir, "main.tf"))                    // id 3
	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputSecond)) // id 4
	wait()

	// var.second leads to variable "second" on lines 3-4
	expectDefinition(5, 3, 4)
}

// The file changes on disk while it is closed, and is opened afterwards.
func TestDidOpen_fileChangedWhileClosedUsesNewContent(t *testing.T) {
	tmpDir := TempDir(t)
	writeReindexFile(t, tmpDir, "variables.tf", reindexVariables)
	writeReindexFile(t, tmpDir, "main.tf", reindexOutputFirst)

	call, wait, expectDefinition, stop := reindexServer(t, tmpDir)
	defer stop()

	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputFirst)) // id 2
	wait()
	call("textDocument/didClose", closeParams(tmpDir, "main.tf")) // id 3
	wait()

	writeReindexFile(t, tmpDir, "main.tf", reindexOutputSecond)
	call("workspace/didChangeWatchedFiles", changedOnDiskParams(tmpDir, "main.tf"))  // id 4
	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputSecond)) // id 5
	wait()

	expectDefinition(6, 3, 4)
}

// Another file in the directory changes on disk while every file is closed,
// and an unchanged file is opened afterwards.
func TestDidOpen_otherFileChangedWhileClosedUsesNewContent(t *testing.T) {
	tmpDir := TempDir(t)
	writeReindexFile(t, tmpDir, "variables.tf", reindexVariables)
	writeReindexFile(t, tmpDir, "main.tf", reindexOutputFirst)

	call, wait, expectDefinition, stop := reindexServer(t, tmpDir)
	defer stop()

	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputFirst)) // id 2
	wait()
	call("textDocument/didClose", closeParams(tmpDir, "main.tf")) // id 3
	wait()

	writeReindexFile(t, tmpDir, "variables.tf", reindexVariablesMoved)
	call("workspace/didChangeWatchedFiles", changedOnDiskParams(tmpDir, "variables.tf")) // id 4
	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputFirst))      // id 5
	wait()

	// var.first now leads to variable "first" on lines 3-4 of the moved file
	expectDefinition(6, 3, 4)
}

// Nothing changes between opens: the definition is found as before.
func TestDidOpen_unchangedReopenStillResolves(t *testing.T) {
	tmpDir := TempDir(t)
	writeReindexFile(t, tmpDir, "variables.tf", reindexVariables)
	writeReindexFile(t, tmpDir, "main.tf", reindexOutputFirst)

	call, wait, expectDefinition, stop := reindexServer(t, tmpDir)
	defer stop()

	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputFirst)) // id 2
	wait()
	call("textDocument/didClose", closeParams(tmpDir, "main.tf"))                   // id 3
	call("textDocument/didOpen", openParams(tmpDir, "main.tf", reindexOutputFirst)) // id 4
	wait()

	// var.first leads to variable "first" on lines 0-1
	expectDefinition(5, 0, 1)
}
