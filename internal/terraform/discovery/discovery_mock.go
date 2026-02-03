// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package discovery

type MockDiscovery struct {
	Path string
}

func (d *MockDiscovery) LookPath() (string, error) {
	return d.Path, nil
}
