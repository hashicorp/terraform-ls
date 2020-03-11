package discovery

import (
	"fmt"
	"os/exec"
)

func LookPath() (string, error) {
	path, err := exec.LookPath("terraform")
	if err != nil {
		return "", fmt.Errorf("unable to find terraform: %s", err)
	}
	return path, nil
}
