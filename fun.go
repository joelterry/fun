/*
Package fun is a library for quickly testing functions.

Example:

	package example

	import (
		"errors"
		"testing"

		"github.com/joelterry/fun"
	)

	func foo(s string) (string, error) {
		if len(s) == 0 {
			panic("expected non-empty string")
		}
		if s[0:3] == "foo" {
			return "", errors.New("you can't foo a foo")
		}
		return s + " foo", nil
	}

	func TestAdd(t *testing.T) {

		f := fun.Test(t, foo)
		f.In("bar").Out("bar foo", nil)
		f.In("foo").Err()
		f.In("").Panic()

		// You can optionally chain the test cases, leave off the last return
		// value if you expect a nil error, and test for specific error/panic values.

		fun.Test(t, foo).
			In("bar").Out("bar foo").
			In("foo").Err(errors.New("you can't foo a foo")).
			In("").Panic("expected non-empty string")

		// As shorthand you can call Err(nil) to check that an error wasn't returned.
		// You can't do the same for Panic, since nil panic values are possible.

		fun.Test(t, foo).In("bar").Err(nil)
		fun.Test(t, foo).In("bar").Panic(nil) // fails
	}

*/
package fun

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

func isError(t reflect.Type) bool {
	errInterface := reflect.TypeOf((*error)(nil)).Elem()
	return t.Implements(errInterface)
}

func trimPkg(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			s = s[i+1:]
			break
		}
	}
	dot := -1
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if c != '.' {
			continue
		}
		if dot == -1 {
			dot = i
			continue
		}
		return s[i+1:]
	}
	return s[dot+1:]
}

// Test is the entry point for testing a function.
// fun must be a func.
func Test(t *testing.T, fun interface{}) *FunTest {
	return test(t, fun)
}

type failer interface {
	Fail()
}

func test(t failer, fun interface{}) *FunTest {
	ft := &FunTest{
		t: t,
	}

	if fun == nil {
		fmt.Printf("Test: 'fun' value passed to Test is nil")
		t.Fail()
		return ft
	}

	val := reflect.ValueOf(fun)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		fmt.Printf("Test: 'fun' value passed to Test isn't a func: %v\n", fun)
		t.Fail()
		return ft
	}

	ft.val = val
	ft.typ = typ
	ft.name = trimPkg(runtime.FuncForPC(val.Pointer()).Name())
	ft.valid = true

	numOut := typ.NumOut()
	ft.errors = numOut > 0 && isError(typ.Out(numOut-1))

	return ft
}

// In is where you pass in the arguments you want to test.
//
// It can either be called from the value returned by Test, or after Out/Err/Panic in a chain.
func (ft *FunTest) In(args ...interface{}) Case {
	ft.i++
	return Case{ft, args}
}

// Out is where you pass in the return variables that you expect.
//
// It should follow In() in a call chain.
//
// If the last return type is an error, and you expect it to be nil, you may leave it out.
func (c Case) Out(results ...interface{}) (ret *FunTest) {
	ret = c.ft

	if !c.ft.valid {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			c.println("panic: ", r)
			c.ft.t.Fail()
		}
	}()

	argVals := make([]reflect.Value, len(c.args))
	for i, arg := range c.args {
		argVals[i] = reflect.ValueOf(arg)
	}
	resVals := c.ft.val.Call(argVals)
	realResults := make([]interface{}, len(resVals))
	for i, resVal := range resVals {
		realResults[i] = resVal.Interface()
	}

	if c.ft.errors && len(results) == len(realResults)-1 {
		last := realResults[len(realResults)-1]
		if last != nil {
			c.println(last)
			c.ft.t.Fail()
			return
		}
		realResults = realResults[:len(realResults)-1]
	}

	if len(realResults) != len(results) {
		c.printf("expected %d results, but got %d\n", len(results), len(realResults))
		c.ft.t.Fail()
		return
	}

	for i, rr := range realResults {
		if !reflect.DeepEqual(rr, results[i]) {
			c.printf("expected (%v), but got (%v)\n", results, realResults)
			c.ft.t.Fail()
			return
		}
	}

	return
}

// Err should be called instead of Out if you just want to check for an error. This is only valid if the tested
// function's final return value is an error.
//
// You can optionally pass in an error if you're expecting something specific.
func (c Case) Err(v ...interface{}) (ret *FunTest) {
	ret = c.ft

	if !c.ft.valid {
		return
	}

	if !c.ft.errors {
		c.println("Err() called with a func that doesn't error")
		c.ft.t.Fail()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			c.println("expected error, but panic occured:", r)
			c.ft.t.Fail()
		}
	}()

	argVals := make([]reflect.Value, len(c.args))
	for i, arg := range c.args {
		argVals[i] = reflect.ValueOf(arg)
	}
	resVals := c.ft.val.Call(argVals)

	if len(resVals) == 0 {
		c.println("expected an error, but no values were returned")
		c.ft.t.Fail()
		return
	}

	last := resVals[len(resVals)-1].Interface()
	err, ok := last.(error)
	if err == nil {
		if len(v) > 0 && v[0] == nil {
			return
		}
		c.println("returned error was not nil")
		c.ft.t.Fail()
		return
	}
	if !ok {
		c.println("last return value was not an error")
		c.ft.t.Fail()
		return
	}

	if len(v) > 0 && !reflect.DeepEqual(v[0], last) {
		c.printf("wrong error: expected %v, but got %v\n", v[0], last)
		c.ft.t.Fail()
		return
	}

	return
}

// Panic should be called instead of Out if you want to check that a panic occured.
//
// You can optionally pass in a value if you're expecting something specific.
func (c Case) Panic(v ...interface{}) (ret *FunTest) {
	ret = c.ft

	if !c.ft.valid {
		return
	}

	didPanic := true

	defer func() {
		if !didPanic {
			return
		}
		r := recover()
		if len(v) == 0 {
			return
		}
		if !reflect.DeepEqual(v[0], r) {
			c.printf("wrong panic value: expected %v, but got %v\n", v[0], r)
			c.ft.t.Fail()
		}
	}()

	argVals := make([]reflect.Value, len(c.args))
	for i, arg := range c.args {
		argVals[i] = reflect.ValueOf(arg)
	}
	c.ft.val.Call(argVals)

	didPanic = false
	c.println("function was called successfully, expected to panic")
	c.ft.t.Fail()

	return
}

// FunTest contains the In method, and can be ignored as a type.
type FunTest struct {
	t      failer
	val    reflect.Value
	typ    reflect.Type
	valid  bool
	errors bool
	name   string
	i      int
}

// Case contains the Out/Err/Panic methods, and can be ignored as a type.
type Case struct {
	ft   *FunTest
	args []interface{}
}

func (c Case) println(a ...interface{}) {
	fmt.Printf("(%s) Case %d: ", c.ft.name, c.ft.i)
	fmt.Println(a...)
}

func (c Case) printf(format string, a ...interface{}) {
	fmt.Printf("(%s) Case %d: ", c.ft.name, c.ft.i)
	fmt.Printf(format, a...)
}
