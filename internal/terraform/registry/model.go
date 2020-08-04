package registry

type Provider struct {
	Attributes ProviderAttr `json:"attributes"`
}

type ProviderAttr struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Tier        string `json:"tier"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type PageLink struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next"`
	Prev  string `json:"prev"`
}

type ProviderPage struct {
	Data  []Provider `json:"data"`
	Links PageLink   `json:"links"`
}
