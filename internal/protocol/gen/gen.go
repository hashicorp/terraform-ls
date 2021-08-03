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
	goplsRef = "gopls/v0.7.0"
	urlFmt   = "https://raw.githubusercontent.com/golang/tools" +
		"/%s/internal/lsp/protocol/tsprotocol.go"
)

func main() {
	args := os.Args[1:]
	if len(args) > 1 && args[0] == "--" {
		args = args[1:]
	}

	if len(args) != 1 {
		log.Fatalf("expected exactly 1 argument (target path), given: %q", args)
	}

	filename, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf(urlFmt, goplsRef)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != 200 {
		log.Fatalf("status code: %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed reading body: %s", err)
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("failed to create file: %s", err)
	}

	n, err := f.Write(b)

	fmt.Printf("%d bytes written to %s\n", n, filename)
}
