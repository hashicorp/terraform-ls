package discovery

import (
	"fmt"
	"os/exec"
)

type DiscoveryFunc func() (string, error)

type Discovery struct{}

func (d *Discovery) LookPath() (string, error) {
	path, err := exec.LookPath(executableName)
	if err != nil {
		return "", fmt.Errorf("unable to find %s: %s", executableName, err)
	}
	return path, nil
}
