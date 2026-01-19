// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/job"
)

type JobIdSliceIndex struct {
	Field string
}

func (s *JobIdSliceIndex) FromObject(obj interface{}) (bool, [][]byte, error) {
	idx := &memdb.StringSliceFieldIndex{Field: s.Field}
	return idx.FromObject(obj)
}

func (s *JobIdSliceIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}
	arg, ok := args[0].(job.ID)
	if !ok {
		return nil, fmt.Errorf("argument must be a job.ID: %#v", args[0])
	}
	// Add the null character as a terminator
	arg += "\x00"
	return []byte(arg), nil
}

func (s *JobIdSliceIndex) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	idx := &memdb.StringSliceFieldIndex{Field: s.Field}
	return idx.PrefixFromArgs(args...)
}
