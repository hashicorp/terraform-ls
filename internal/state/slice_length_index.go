// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-memdb"
)

type SliceLengthIndex struct {
	Field string
}

func (s *SliceLengthIndex) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(s.Field)
	if !fv.IsValid() {
		return false, nil,
			fmt.Errorf("field '%s' for %#v is invalid", s.Field, obj)
	}

	// Check the type
	k := fv.Kind()
	if k != reflect.Slice {
		return false, nil, fmt.Errorf("field %q is of type %v; want a slice", s.Field, k)
	}

	// Get the slice length and encode it
	val := fv.Len()
	buf := encodeInt(int64(val), 8)

	return true, buf, nil
}

func (s *SliceLengthIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}

	v := reflect.ValueOf(args[0])
	if !v.IsValid() {
		return nil, fmt.Errorf("%#v is invalid", args[0])
	}

	k := v.Kind()
	_, ok := memdb.IsIntType(k)
	if !ok {
		return nil, fmt.Errorf("arg is of type %v; want an int", k)
	}

	val := v.Int()
	buf := encodeInt(val, 8)

	return buf, nil
}
