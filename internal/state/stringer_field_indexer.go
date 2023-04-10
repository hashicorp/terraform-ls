// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"bytes"
	"fmt"
	"reflect"
)

type stringer interface {
	String() string
}

type StringerFieldIndexer struct {
	Field string
}

func (s *StringerFieldIndexer) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(s.Field)

	strMethod := fv.MethodByName("String")

	if !strMethod.IsValid() {
		return false, nil, fmt.Errorf("%q: not indexable as string", s.Field)
	}

	val := strMethod.Call([]reflect.Value{})

	if len(val) != 1 {
		return false, nil, fmt.Errorf("%q: not indexable as string", s.Field)
	}

	value := val[0].String()
	if value == "" {
		return false, nil, nil
	}

	// Add the null character as a terminator
	value += "\x00"
	return true, []byte(value), nil
}

func (s *StringerFieldIndexer) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	idx, err := s.FromArgs(args...)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(idx, []byte("\x00")), nil
}

func (s *StringerFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}
	arg, ok := args[0].(stringer)
	if !ok {
		return nil, fmt.Errorf("argument must be convertible to string: %#v", args[0])
	}

	val := arg.String()
	// Add the null character as a terminator
	val += "\x00"

	return []byte(val), nil
}
