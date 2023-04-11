// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

type MockDiscovery struct {
	Path string
}

func (d *MockDiscovery) LookPath() (string, error) {
	return d.Path, nil
}
