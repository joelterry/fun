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

func Test(t *testing.T, fun interface{}) *FunTest {
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

func (ft *FunTest) In(args ...interface{}) Case {
	ft.i++
	return Case{ft, args}
}

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

func (c Case) Err() (ret *FunTest) {
	ret = c.ft

	if !c.ft.valid {
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
	if !ok {
		c.println("last return value was not an error")
		c.ft.t.Fail()
		return
	}
	if err == nil {
		c.println("returned error was nil")
		c.ft.t.Fail()
		return
	}

	return
}

func (c Case) Panic() (ret *FunTest) {
	ret = c.ft

	if !c.ft.valid {
		return
	}

	defer func() {
		recover()
	}()

	argVals := make([]reflect.Value, len(c.args))
	for i, arg := range c.args {
		argVals[i] = reflect.ValueOf(arg)
	}
	c.ft.val.Call(argVals)

	c.println("function was called successfully, expected to panic")
	c.ft.t.Fail()

	return
}

type FunTest struct {
	t      *testing.T
	val    reflect.Value
	typ    reflect.Type
	valid  bool
	errors bool
	name   string
	i      int
}

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
