package schemas

import "github.com/hashicorp/go-version"

type RawVersionOutput struct {
	CoreVersion string            `json:"core"`
	Providers   map[string]string `json:"providers"`
}

type VersionOutput struct {
	Core      *version.Version
	Providers map[string]*version.Version
}
