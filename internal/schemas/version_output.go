package schemas

type VersionOutput struct {
	CoreVersion string            `json:"core"`
	Providers   map[string]string `json:"providers"`
}
