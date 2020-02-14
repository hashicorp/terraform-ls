package filesystem

type dir struct {
	files map[string]*file
}

func newDir() *dir {
	return &dir{
		files: make(map[string]*file),
	}
}
