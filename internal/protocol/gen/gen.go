// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build generate
// +build generate

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	goplsRef = "gopls/v0.10.0"
	urlFmt   = "https://raw.githubusercontent.com/golang/tools" +
		"/%s/gopls/internal/lsp/protocol/%s"
)

func main() {
	args := os.Args[1:]
	if len(args) > 1 && args[0] == "--" {
		args = args[1:]
	}

	if len(args) != 2 {
		log.Fatalf("expected exactly 2 arguments (source filename & target path), given: %q", args)
	}

	sourceFilename := args[0]

	targetFilename, err := filepath.Abs(args[1])
	if err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf(urlFmt, goplsRef, sourceFilename)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != 200 {
		log.Fatalf("status code: %d (%s)", resp.StatusCode, url)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed reading body: %s", err)
	}

	f, err := os.Create(targetFilename)
	if err != nil {
		log.Fatalf("failed to create file: %s", err)
	}

	n, err := f.Write(b)

	fmt.Printf("%d bytes written to %s\n", n, targetFilename)
}
