// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"
)

// See https://github.com/hashicorp/go-memdb/pull/117

// TimeFieldIndex is used to extract a time.Time field from an object using
// reflection and builds an index on that field.
type TimeFieldIndex struct {
	Field string
}

func (u *TimeFieldIndex) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(u.Field)
	if !fv.IsValid() {
		return false, nil,
			fmt.Errorf("field '%s' for %#v is invalid", u.Field, obj)
	}

	// Check the type
	k := fv.Kind()

	if ok := IsTimeType(k); !ok {
		return false, nil, fmt.Errorf("field %q is of type %v; want a time."+
			"Time", u.Field, k)
	}

	// Get the value and encode it
	val := fv.Interface().(time.Time)
	bufUnix := encodeInt(val.Unix(), 8)
	bufNano := encodeInt(int64(val.Nanosecond()), 4)
	buf := append(bufUnix, bufNano...)

	return true, buf, nil
}

func (u *TimeFieldIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}

	v := reflect.ValueOf(args[0])
	if !v.IsValid() {
		return nil, fmt.Errorf("%#v is invalid", args[0])
	}

	k := v.Kind()

	if ok := IsTimeType(k); !ok {
		return nil, fmt.Errorf("arg is of type %v; want a time.Time", k)
	}

	val := v.Interface().(time.Time)
	bufUnix := encodeInt(val.Unix(), 8)
	bufNano := encodeInt(int64(val.Nanosecond()), 4)
	buf := append(bufUnix, bufNano...)

	return buf, nil
}

func encodeInt(val int64, size int) []byte {
	buf := make([]byte, size)

	// This bit flips the sign bit on any sized signed twos-complement integer,
	// which when truncated to a uint of the same size will bias the value such
	// that the maximum negative int becomes 0, and the maximum positive int
	// becomes the maximum positive uint.
	scaled := val ^ int64(-1<<(size*8-1))

	switch size {
	case 1:
		buf[0] = uint8(scaled)
	case 2:
		binary.BigEndian.PutUint16(buf, uint16(scaled))
	case 4:
		binary.BigEndian.PutUint32(buf, uint32(scaled))
	case 8:
		binary.BigEndian.PutUint64(buf, uint64(scaled))
	default:
		panic(fmt.Sprintf("unsupported int size parameter: %d", size))
	}

	return buf
}

// IsTimeType returns whether the passed type is a type of time.Time.
func IsTimeType(k reflect.Kind) (okay bool) {
	switch k {
	case reflect.TypeOf(time.Time{}).Kind():
		return true
	default:
		return false
	}
}
