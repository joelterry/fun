package fun

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func isError(t reflect.Type) bool {
	errInterface := reflect.TypeOf((*error)(nil)).Elem()
	return t.Implements(errInterface)
}

func Test(t *testing.T, fun interface{}) FunTest {
	val := reflect.ValueOf(fun)
	typ := val.Type()
	valid := typ.Kind() == reflect.Func
	errors := false
	if valid {
		numOut := typ.NumOut()
		errors = numOut > 0 && isError(typ.Out(numOut-1))
	} else {
		fmt.Println("Test: 'fun' value passed to Test isn't a func")
		t.Fail()
	}
	return FunTest{
		t:      t,
		fun:    fun,
		val:    val,
		typ:    typ,
		valid:  valid,
		errors: errors,
		i:      1,
	}
}

func (ft FunTest) In(args ...interface{}) Case {
	return Case{ft, args}
}

func (c Case) Out(results ...interface{}) (ret FunTest) {
	ret = c.ft
	ret.i++

	if !c.ft.valid {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			c.println(r)
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

func (c Case) Err() (ret FunTest) {
	ret = c.ft
	ret.i++

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

func (c Case) Panic() (ret FunTest) {
	ret = c.ft
	ret.i++

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
	fun    interface{}
	val    reflect.Value
	typ    reflect.Type
	valid  bool
	errors bool
	i      int
}

type Case struct {
	ft   FunTest
	args []interface{}
}

func (c Case) println(a ...interface{}) {
	fmt.Print("Case ", strconv.Itoa(c.ft.i), ": ")
	fmt.Println(a...)
}

func (c Case) printf(format string, a ...interface{}) {
	fmt.Print("Case ", strconv.Itoa(c.ft.i), ": ")
	fmt.Printf(format, a...)
}
