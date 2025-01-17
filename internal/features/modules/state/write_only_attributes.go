// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

type ResourceName = string
type AttributeName = string

type WriteOnlyAttributes map[tfaddr.Provider]map[ResourceName]map[AttributeName]int
