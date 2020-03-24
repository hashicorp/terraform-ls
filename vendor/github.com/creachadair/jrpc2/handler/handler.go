// Package handler provides implementations of the jrpc2.Assigner interface,
// and support for adapting functions to the jrpc2.Handler interface.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"bitbucket.org/creachadair/stringset"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
)

// A Func adapts a function having the correct signature to a jrpc2.Handler.
type Func func(context.Context, *jrpc2.Request) (interface{}, error)

// Handle implements the jrpc2.Handler interface by calling m.
func (m Func) Handle(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
	return m(ctx, req)
}

// A Map is a trivial implementation of the jrpc2.Assigner interface that looks
// up method names in a map of static jrpc2.Handler values.
type Map map[string]jrpc2.Handler

// Assign implements part of the jrpc2.Assigner interface.
func (m Map) Assign(_ context.Context, method string) jrpc2.Handler { return m[method] }

// Names implements part of the jrpc2.Assigner interface.
func (m Map) Names() []string { return stringset.FromKeys(m).Elements() }

// A ServiceMap combines multiple assigners into one, permitting a server to
// export multiple services under different names.
//
// Example:
//    m := handler.ServiceMap{
//      "Foo": jrpc2.NewService(fooService),  // methods Foo.A, Foo.B, etc.
//      "Bar": jrpc2.NewService(barService),  // methods Bar.A, Bar.B, etc.
//    }
//
type ServiceMap map[string]jrpc2.Assigner

// Assign splits the inbound method name as Service.Method, and passes the
// Method portion to the corresponding Service assigner. If method does not
// have the form Service.Method, or if Service is not set in m, the lookup
// fails and returns nil.
func (m ServiceMap) Assign(ctx context.Context, method string) jrpc2.Handler {
	parts := strings.SplitN(method, ".", 2)
	if len(parts) == 1 {
		return nil
	} else if ass, ok := m[parts[0]]; ok {
		return ass.Assign(ctx, parts[1])
	}
	return nil
}

// Names reports the composed names of all the methods in the service, each
// having the form Service.Method.
func (m ServiceMap) Names() []string {
	var all stringset.Set
	for svc, assigner := range m {
		for _, name := range assigner.Names() {
			all.Add(svc + "." + name)
		}
	}
	return all.Elements()
}

// New adapts a function to a jrpc2.Handler. The concrete value of fn must be a
// function with one of the following type signatures:
//
//    func(context.Context) error
//    func(context.Context) Y
//    func(context.Context) (Y, error)
//    func(context.Context, X) error
//    func(context.Context, X) Y
//    func(context.Context, X) (Y, error)
//    func(context.Context, ...X) (Y, error)
//    func(context.Context, *jrpc2.Request) (Y, error)
//    func(context.Context, *jrpc2.Request) (interface{}, error)
//
// for JSON-marshalable types X and Y. New will panic if the type of fn does
// not have one of these forms.  The resulting method will handle encoding and
// decoding of JSON and report appropriate errors.
//
// Functions adapted by in this way can obtain the *jrpc2.Request value using
// the jrpc2.InboundRequest helper on the context value supplied by the server.
func New(fn interface{}) Func {
	m, err := newHandler(fn)
	if err != nil {
		panic(err)
	}
	return m
}

// NewService adapts the methods of a value to a map from method names to
// Handler implementations as constructed by New. It will panic if obj has no
// exported methods with a suitable signature.
func NewService(obj interface{}) Map {
	out := make(Map)
	val := reflect.ValueOf(obj)
	typ := val.Type()

	// This considers only exported methods, as desired.
	for i, n := 0, val.NumMethod(); i < n; i++ {
		mi := val.Method(i)
		if v, err := newHandler(mi.Interface()); err == nil {
			out[typ.Method(i).Name] = v
		}
	}
	if len(out) == 0 {
		panic("no matching exported methods")
	}
	return out
}

var (
	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem() // type context.Context
	errType = reflect.TypeOf((*error)(nil)).Elem()           // type error
	reqType = reflect.TypeOf((*jrpc2.Request)(nil))          // type *jrpc2.Request
)

func newHandler(fn interface{}) (Func, error) {
	if fn == nil {
		return nil, errors.New("nil method")
	}

	// Special case: If fn has the exact signature of the Handle method, don't do
	// any (additional) reflection at all.
	if f, ok := fn.(func(context.Context, *jrpc2.Request) (interface{}, error)); ok {
		return Func(f), nil
	}

	// Check that fn is a function of one of the correct forms.
	typ, err := checkFunctionType(fn)
	if err != nil {
		return nil, err
	}

	// Construct a function to unpack the parameters from the request message,
	// based on the signature of the user's callback.
	var newinput func(req *jrpc2.Request) ([]reflect.Value, error)

	if typ.NumIn() == 1 {
		// Case 1: The function does not want any request parameters.
		// Nothing needs to be decoded, but verify no parameters were passed.
		newinput = func(req *jrpc2.Request) ([]reflect.Value, error) {
			if req.HasParams() {
				return nil, jrpc2.Errorf(code.InvalidParams, "no parameters accepted")
			}
			return nil, nil
		}

	} else if a := typ.In(1); a == reqType {
		// Case 2: The function wants the underlying *jrpc2.Request value.
		newinput = func(req *jrpc2.Request) ([]reflect.Value, error) {
			return []reflect.Value{reflect.ValueOf(req)}, nil
		}

	} else {
		// Check whether the function wants a pointer to its argument.  We need
		// to create one either way to support unmarshaling, but we need to
		// indirect it back off if the callee didn't want it.

		// Case 3a: The function wants a bare value, not a pointer.
		argType := typ.In(1)
		undo := reflect.Value.Elem

		if argType.Kind() == reflect.Ptr {
			// Case 3b: The function wants a pointer.
			undo = func(v reflect.Value) reflect.Value { return v }
			argType = argType.Elem()
		}

		newinput = func(req *jrpc2.Request) ([]reflect.Value, error) {
			in := reflect.New(argType).Interface()
			if err := req.UnmarshalParams(in); err != nil {
				return nil, jrpc2.Errorf(code.InvalidParams, "invalid parameters: %v", err)
			}
			arg := reflect.ValueOf(in)
			return []reflect.Value{undo(arg)}, nil
		}
	}

	// Construct a function to decode the result values.
	var decodeOut func([]reflect.Value) (interface{}, error)

	switch typ.NumOut() {
	case 1:
		if typ.Out(0) == errType {
			// A function that returns only error: Result is always nil.
			decodeOut = func(vals []reflect.Value) (interface{}, error) {
				oerr := vals[0].Interface()
				if oerr != nil {
					return nil, oerr.(error)
				}
				return nil, nil
			}
		} else {
			// A function that returns a single non-error: err is always nil.
			decodeOut = func(vals []reflect.Value) (interface{}, error) {
				return vals[0].Interface(), nil
			}
		}
	default:
		// A function that returns a value and an error.
		decodeOut = func(vals []reflect.Value) (interface{}, error) {
			out, oerr := vals[0].Interface(), vals[1].Interface()
			if oerr != nil {
				return nil, oerr.(error)
			}
			return out, nil
		}
	}

	f := reflect.ValueOf(fn)
	call := f.Call
	if typ.IsVariadic() {
		call = f.CallSlice
	}

	return Func(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		rest, ierr := newinput(req)
		if ierr != nil {
			return nil, ierr
		}
		args := append([]reflect.Value{reflect.ValueOf(ctx)}, rest...)
		return decodeOut(call(args))
	}), nil
}

func checkFunctionType(fn interface{}) (reflect.Type, error) {
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, errors.New("not a function")
	} else if np := typ.NumIn(); np == 0 || np > 2 {
		return nil, errors.New("wrong number of parameters")
	} else if no := typ.NumOut(); no < 1 || no > 2 {
		return nil, errors.New("wrong number of results")
	} else if typ.In(0) != ctxType {
		return nil, errors.New("first parameter is not context.Context")
	} else if no == 2 && typ.Out(1) != errType {
		return nil, errors.New("result is not of type error")
	}
	return typ, nil
}

// Args is a wrapper that decodes an array of positional parameters into
// concrete locations.
//
// Unmarshaling a JSON value into an Args value v succeeds if the JSON encodes
// an array with length len(v), and unmarshaling each subvalue i into the
// corresponding v[i] succeeds.  As a special case, if v[i] == nil the
// corresponding value is discarded.
//
// Marshaling an Args value v into JSON succeeds if each element of the slice
// is JSON marshalable, and yields a JSON array of length len(v) containing the
// JSON values corresponding to the elements of v.
//
// Usage example:
//
//    func Handler(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
//       var x, y int
//       var s string
//
//       if err := req.UnmarshalParams(&handler.Args{&x, &y, &s}); err != nil {
//          return nil, err
//       }
//       // do useful work with x, y, and s
//    }
//
type Args []interface{}

// UnmarshalJSON supports JSON unmarshaling for a.
func (a Args) UnmarshalJSON(data []byte) error {
	var elts []json.RawMessage
	if err := json.Unmarshal(data, &elts); err != nil {
		return fmt.Errorf("decoding args: %w", err)
	} else if len(elts) != len(a) {
		return fmt.Errorf("wrong number of args (got %d, want %d)", len(elts), len(a))
	}
	for i, elt := range elts {
		if a[i] == nil {
			continue
		} else if err := json.Unmarshal(elt, a[i]); err != nil {
			return fmt.Errorf("decoding argument %d: %w", i+1, err)
		}
	}
	return nil
}

// MarshalJSON supports JSON marshaling for a.
func (a Args) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte(`[]`), nil
	}
	return json.Marshal([]interface{}(a))
}

// Obj is a wrapper that decodes object fields into concrete locations.
//
// Unmarshaling a JSON text into an Obj value v succeeds if the JSON encodes an
// object, and unmarshaling the value for each key k of the object into v[k]
// succeeds. If k does not exist in v, it is ignored.
//
// Usage example:
//
//    var x, y int
//    var s string
//
//    if err := req.UnmarshalParams(handler.Obj{
//       "left":  &x,
//       "right": &x,
//       "tag":   &s,
//    }); err != nil {
//       return nil, err
//    }
//    // do useful work with x, y, and s
//
type Obj map[string]interface{}

// UnmarshalJSON supports JSON unmarshaling into o.
func (o Obj) UnmarshalJSON(data []byte) error {
	var base map[string]json.RawMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return fmt.Errorf("decoding object: %v", err)
	}
	for key, val := range base {
		arg, ok := o[key]
		if !ok {
			continue
		} else if err := json.Unmarshal(val, arg); err != nil {
			return fmt.Errorf("decoding %q: %v", key, err)
		}
	}
	return nil
}
