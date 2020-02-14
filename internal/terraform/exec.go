package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

type terraformExec struct {
	workDir string
	Logger  *log.Logger
}

func TerraformExec(workdir string) *terraformExec {
	return &terraformExec{workDir: workdir}
}

func (te *terraformExec) run(args ...string) ([]byte, error) {
	allArgs := []string{"terraform"}
	allArgs = append(allArgs, args...)

	var outBuf bytes.Buffer
	var errBuf strings.Builder

	path, err := exec.LookPath("terraform")
	if err != nil {
		log.Printf("Current PATH: %q", os.Getenv("PATH"))
		return nil, fmt.Errorf("unable to find terraform for %q: %s", te.workDir, err)
	}

	cmd := &exec.Cmd{
		Path:   path,
		Args:   allArgs,
		Dir:    te.workDir,
		Stderr: &errBuf,
		Stdout: &outBuf,
	}
	err = cmd.Run()
	if err != nil {
		if tErr, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("terraform failed: %s\n\nstderr:\n%s", tErr.ProcessState.String(), errBuf.String())
		}
		return nil, err
	}

	return outBuf.Bytes(), nil
}

func (te *terraformExec) Version() (string, error) {
	out, err := te.run("version")
	if err != nil {
		return "", fmt.Errorf("failed to get version: %s", err)
	}

	return string(out), nil
}

func (te *terraformExec) ProviderSchemas() (*tfjson.ProviderSchemas, error) {
	outBytes, err := te.run("providers", "schema", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %s", err)
	}

	var schemas tfjson.ProviderSchemas
	err = json.Unmarshal(outBytes, &schemas)
	if err != nil {
		return nil, err
	}

	return &schemas, nil
}
