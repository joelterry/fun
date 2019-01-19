package fun

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

type testFailer struct {
	failed bool
}

func (tf *testFailer) Fail() {
	tf.failed = true
}

func sumUnder10(ns ...int) (int, error) {
	r := 0
	for _, n := range ns {
		if n < 0 {
			panic(strconv.Itoa(n) + " is negative")
		}
		r += n
		if r >= 10 {
			return 0, errors.New("sum should be less than 10")
		}
	}
	return r, nil
}

func TestMain(t *testing.T) {
	tf := &testFailer{}
	f := test(tf, sumUnder10)

	if tf.failed {
		t.FailNow()
	}

	passCases := []func(){
		func() { f.In(1, 2).Out(3) },
		func() { f.In(1, 2).Out(3, nil) },
		func() { f.In(1, 2, 3).Out(6) },
		func() { f.In(5, 5).Out(0, errors.New("sum should be less than 10")) },
		func() { f.In(5, 5).Err() },
		func() { f.In(4, 4).Err(nil) },
		func() { f.In(5, 5).Err(errors.New("sum should be less than 10")) },
		func() { f.In(-1, 2, 3).Panic() },
		func() { f.In(-1, 2, 3).Panic("-1 is negative") },
	}

	for i, c := range passCases {
		c()
		if tf.failed {
			fmt.Println("pass case", i+1, "failed")
			t.Fail()
			tf.failed = false
		}
	}

	failCases := []func(){
		func() { f.In().Out() },
		func() { f.In(-1).Out(-1) },
		func() { f.In(4, 4).Out(10) },
		func() { f.In(5, 5).Out(10) },
		func() { f.In(1, 2).Err() },
		func() { f.In(5, 5).Err(nil) },
		func() { f.In(-1).Err() },
		func() { f.In(5, 5).Err(errors.New("wrong error")) },
		func() { f.In(1, -2, 3).Panic("-1 is negative") },
		func() { f.In(1).Panic() },
	}

	for i, c := range failCases {
		c()
		if !tf.failed {
			fmt.Println("fail case", i+1, "failed")
			t.Fail()
		}
		tf.failed = false
	}

	invalid := test(tf, "string")
	if !tf.failed {
		t.Fail()
	}
	invalidCases := []func(){
		func() { invalid.In().Out() },
		func() { invalid.In().Err() },
		func() { invalid.In().Panic() },
	}
	// these cases are early returns
	// rather than extra failures
	for i, c := range invalidCases {
		c()
		if !tf.failed {
			fmt.Println("invalid case", i+1, "failed")
			t.Fail()
		}
	}

	// nil invalid case
	tf.failed = false
	invalid = test(tf, nil)
	if !tf.failed {
		t.Fail()
	}

	tf.failed = false
	novalues := test(tf, func() {})
	novalues.In().Out()
	if tf.failed {
		t.Fail()
		tf.failed = false
	}
	failCases = []func(){
		func() { novalues.In(1).Out() },
		func() { novalues.In().Err() },
		func() { novalues.In().Panic() },
	}
	for i, c := range failCases {
		c()
		if !tf.failed {
			fmt.Println("no values case", i+1, "failed")
			t.Fail()
		}
		tf.failed = false
	}
}

// The rest of the tests use the fun library,
// so they're dependent on TestMain passing.

func TestIsError(t *testing.T) {
	Test(t, isError).
		In(nil).Panic().
		In(reflect.TypeOf(nil)).Panic().
		In(reflect.TypeOf(errors.New(""))).Out(true)
}

func TestTrimPkg(t *testing.T) {
	x := Test(t, trimPkg)
	x.In("pkg.Func").Out("Func")
	x.In("pkg.Type.Func").Out("Type.Func")
	x.In("github.com/author/pkg.Func").Out("Func")
	x.In("github.com/author/pkg.Type.Func").Out("Type.Func")
	x.In("github.com/author/pkg.(*Type).Func").Out("(*Type).Func")
	x.In("").Out("")
}
