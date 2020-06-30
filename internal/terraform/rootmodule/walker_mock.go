package rootmodule

func MockWalker() *Walker {
	w := NewWalker()
	w.sync = true
	return w
}
