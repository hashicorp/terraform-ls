// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/job"
)

type JobPriorityIndex struct {
	PriorityIntField   string
	IsDirOpenBoolField string
}

func (jpi *JobPriorityIndex) FromObject(obj interface{}) (bool, []byte, error) {
	prioField, size, err := getIntField(obj, jpi.PriorityIntField)
	if err != nil {
		return false, []byte{}, err
	}
	if !prioField.IsZero() {
		// Get the value and encode it
		val := prioField.Int()
		buf := encodeInt(val, size)
		return true, buf, nil
	}

	// Where explicit priority is not set
	// imply it from IsDirOpenBoolField
	isDirOpenField, err := getBoolField(obj, jpi.IsDirOpenBoolField)
	if err != nil {
		return false, []byte{}, err
	}
	impliedPriority := job.LowPriority
	if isDirOpenField.Bool() {
		impliedPriority = job.HighPriority
	}

	buf := encodeInt(int64(impliedPriority), size)
	return true, buf, nil
}

func getIntField(obj interface{}, fieldName string) (reflect.Value, int, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fieldValue := v.FieldByName(fieldName)
	if !fieldValue.IsValid() {
		return reflect.Value{}, 0, fmt.Errorf("field '%s' for %#v is invalid", fieldName, obj)
	}

	// Check the type
	k := fieldValue.Kind()
	size, ok := memdb.IsIntType(k)
	if !ok {
		return reflect.Value{}, 0, fmt.Errorf("field %q is of type %v; want an int", fieldName, k)
	}

	return fieldValue, size, nil
}

func getBoolField(obj interface{}, fieldName string) (reflect.Value, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fieldValue := v.FieldByName(fieldName)
	if !fieldValue.IsValid() {
		return reflect.Value{}, fmt.Errorf("field '%s' for %#v is invalid", fieldName, obj)
	}

	// Check the type
	k := fieldValue.Kind()
	if k != reflect.Bool {
		return reflect.Value{}, fmt.Errorf("field %q is of type %v; want a bool", fieldName, k)
	}

	return fieldValue, nil
}

func (jpi *JobPriorityIndex) FromArgs(args ...interface{}) ([]byte, error) {
	intIdx := &memdb.IntFieldIndex{}
	return intIdx.FromArgs(args...)
}
