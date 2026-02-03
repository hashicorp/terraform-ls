// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-ls/internal/document"
)

type DirHandleFieldIndexer struct {
	Field string
}

func (s *DirHandleFieldIndexer) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(s.Field)
	isPtr := fv.Kind() == reflect.Ptr
	rawHandle := fv.Interface()
	if rawHandle == nil {
		return false, nil, nil
	}

	dh, ok := rawHandle.(document.DirHandle)
	if !ok {
		return false, nil,
			fmt.Errorf("field '%s' for %#v is invalid %v ", s.Field, obj, isPtr)
	}

	val := dh.URI
	if val == "" {
		return false, nil, nil
	}

	// Add the null character as a terminator
	val += "\x00"

	return true, []byte(val), nil
}

func (s *DirHandleFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}
	if args[0] == nil {
		return nil, nil
	}
	arg, ok := args[0].(document.DirHandle)
	if !ok {
		return nil, fmt.Errorf("argument must be a DirHandle: %#v", args[0])
	}

	val := arg.URI
	// Add the null character as a terminator
	val += "\x00"

	return []byte(val), nil
}

func (s *DirHandleFieldIndexer) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	idx, err := s.FromArgs(args...)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(idx, []byte("\x00")), nil
}
