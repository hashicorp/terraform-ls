// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

type InstalledProviders map[tfaddr.Provider]*version.Version

func (ip InstalledProviders) Equals(p InstalledProviders) bool {
	if len(ip) != len(p) {
		return false
	}

	for pAddr, ver := range ip {
		c, ok := p[pAddr]
		if !ok {
			return false
		}
		if !ver.Equal(c) {
			return false
		}
	}

	return true
}
