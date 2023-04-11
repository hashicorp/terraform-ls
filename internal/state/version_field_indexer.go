// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/hashicorp/go-version"
)

type VersionFieldIndexer struct {
	Field string
}

func (s *VersionFieldIndexer) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(s.Field)
	isPtr := fv.Kind() == reflect.Ptr
	rawVersion := fv.Interface()
	if rawVersion == nil {

		return false, nil, nil
	}

	ver, ok := rawVersion.(*version.Version)
	if !ok {
		return false, nil,
			fmt.Errorf("field '%s' for %#v is invalid %v ", s.Field, obj, isPtr)
	}
	if ver == nil {
		return false, nil, nil
	}

	val := ver.String()
	if val == "" {
		return false, nil, nil
	}

	// Add the null character as a terminator
	val += "\x00"

	return true, []byte(val), nil
}

func (s *VersionFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}
	if args[0] == nil {
		return nil, nil
	}
	arg, ok := args[0].(*version.Version)
	if !ok {
		return nil, fmt.Errorf("argument must be a version: %#v", args[0])
	}
	if arg == nil {
		return nil, nil
	}

	val := arg.String()
	// Add the null character as a terminator
	val += "\x00"

	return []byte(val), nil
}

func (s *VersionFieldIndexer) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	idx, err := s.FromArgs(args...)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(idx, []byte("\x00")), nil
}
