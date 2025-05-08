package test

import (
	"fmt"
	"testing"
)

const (
	RecorderFailNow int = iota
)

type recorder struct {
	testing.TB
	fail   func(string)
	fatal  func(string)
	failed bool
}

func (r *recorder) Error(args ...any) {
	r.failed = true
	r.fail(fmt.Sprint(args...))
}

func (r *recorder) Errorf(format string, args ...any) {
	r.failed = true
	r.fail(fmt.Sprintf(format, args...))
}

func (r *recorder) Fail() {
	r.failed = true
}

func (r *recorder) FailNow() {
	r.failed = true
	panic(RecorderFailNow)
}

func (r *recorder) Failed() bool {
	return r.failed
}

func (r *recorder) Fatal(args ...any) {
	r.failed = true
	r.fatal(fmt.Sprint(args...))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.failed = true
	r.fatal(fmt.Sprintf(format, args...))
}
