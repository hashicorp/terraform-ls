package lsp

// LanguageID represents the coding language
// of a file
type LanguageID string

const (
	Terraform LanguageID = "terraform"
	Tfvars    LanguageID = "terraform-vars"
)

func (l LanguageID) String() string {
	return string(l)
}
