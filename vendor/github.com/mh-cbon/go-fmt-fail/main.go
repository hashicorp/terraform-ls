package main

import (
	"os"
	"os/exec"

	"github.com/mh-cbon/go-fmt-fail/hasWritten"
)

func main() {
	args := os.Args

	bin := "go"
	args = append([]string{"fmt"}, args[1:]...)

	stdout := hasWritten.New(os.Stdout)
	stderr := hasWritten.New(os.Stderr)
	oCmd := exec.Command(bin, args...)
	oCmd.Stdout = stdout
	oCmd.Stderr = stderr
	err := oCmd.Run()

	if err != nil {
		os.Exit(1)
	}
	if stdout.Written || stderr.Written {
		os.Exit(1)
	}
}
