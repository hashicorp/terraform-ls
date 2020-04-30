package discovery

type MockDiscovery struct {
	Path string
}

func (d *MockDiscovery) LookPath() (string, error) {
	return d.Path, nil
}
