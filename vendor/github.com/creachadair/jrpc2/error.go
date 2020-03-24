package jrpc2

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/creachadair/jrpc2/code"
)

// Error is the concrete type of errors returned from RPC calls.
type Error struct {
	message string
	code    code.Code
	data    json.RawMessage
}

// Error renders e to a human-readable string for the error interface.
func (e Error) Error() string { return fmt.Sprintf("[%d] %s", e.code, e.message) }

// Code returns the error code value associated with e.
func (e Error) Code() code.Code { return e.code }

// Message returns the message string associated with e.
func (e Error) Message() string { return e.message }

// HasData reports whether e has error data to unmarshal.
func (e Error) HasData() bool { return len(e.data) != 0 }

// UnmarshalData decodes the error data associated with e into v.  It returns
// ErrNoData without modifying v if there was no data message attached to e.
func (e Error) UnmarshalData(v interface{}) error {
	if !e.HasData() {
		return ErrNoData
	}
	return json.Unmarshal([]byte(e.data), v)
}

// MarshalJSON implements the json.Marshaler interface for Error values.
func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(jerror{C: int32(e.code), M: e.message, D: e.data})
}

// UnmarshalJSON implements the json.Unmarshaler interface for Error values.
func (e *Error) UnmarshalJSON(data []byte) error {
	var v jerror
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	e.code = code.Code(v.C)
	e.message = v.M
	e.data = v.D
	return nil
}

// ErrNoData indicates that there are no data to unmarshal.
var ErrNoData = errors.New("no data to unmarshal")

// errServerStopped is returned by Server.Wait when the server was shut down by
// an explicit call to its Stop method or orderly termination of its channel.
var errServerStopped = errors.New("the server has been stopped")

// errClientStopped is the error reported when a client is shut down by an
// explicit call to its Close method.
var errClientStopped = errors.New("the client has been stopped")

// ErrConnClosed is returned by a server's Push method if it is called after
// the client connection is closed.
var ErrConnClosed = errors.New("client connection is closed")

// Errorf returns an error value of concrete type *Error having the specified
// code and formatted message string.
// It is shorthand for DataErrorf(code, nil, msg, args...)
func Errorf(code code.Code, msg string, args ...interface{}) error {
	return DataErrorf(code, nil, msg, args...)
}

// DataErrorf returns an error value of concrete type *Error having the
// specified code, error data, and formatted message string.
// If v == nil this behaves identically to Errorf(code, msg, args...).
func DataErrorf(code code.Code, v interface{}, msg string, args ...interface{}) error {
	e := &Error{code: code, message: fmt.Sprintf(msg, args...)}
	if v != nil {
		if data, err := json.Marshal(v); err == nil {
			e.data = data
		}
	}
	return e
}
