package schemas

import "github.com/hashicorp/go-version"

type VersionOutput struct {
	CoreVersion string            `json:"core"`
	Providers   map[string]string `json:"providers"`
}

type Version struct {
	Core      *version.Version
	Providers map[string]*version.Version
}
