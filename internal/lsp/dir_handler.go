package lsp

type DirHandler interface {
	Dir() string
	URI() string
}
