package registry

type ModuleResponse struct {
	Version string     `json:"version"`
	Root    ModuleRoot `json:"root"`
}

type ModuleRoot struct {
	Inputs  []Input  `json:"inputs"`
	Outputs []Output `json:"outputs"`
}

type Input struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
}

type Output struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ModuleVersionsResponse struct {
	Modules []ModuleVersionsEntry `json:"modules"`
}

type ModuleVersionsEntry struct {
	Versions []ModuleVersion `json:"versions"`
}

type ModuleVersion struct {
	Version string `json:"version"`
}
