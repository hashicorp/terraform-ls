// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package job

type ID string

func (id ID) String() string {
	return string(id)
}

type IDs []ID

func (ids IDs) Copy() IDs {
	newIds := make([]ID, len(ids))

	for i, id := range ids {
		newIds[i] = id
	}

	return newIds
}

func (ids IDs) StringSlice() []string {
	stringIds := make([]string, len(ids))

	for i, id := range ids {
		stringIds[i] = id.String()
	}

	return stringIds
}
