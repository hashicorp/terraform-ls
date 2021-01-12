package module

func MockWalker() *Walker {
	w := NewWalker()
	w.sync = true
	return w
}
