package test

import (
	"fmt"
	"testing"
)

const (
	RecorderFailNow int = iota
	RecorderFatal
	RecorderFatalf
)

type recorder struct {
	testing.TB
	fail   func(string)
	fatal  func(string)
	failed bool
}

func (r *recorder) Error(args ...any) {
	r.Helper()
	r.failed = true
	r.fail(fmt.Sprint(args...))
}

func (r *recorder) Errorf(format string, args ...any) {
	r.Helper()
	r.failed = true
	r.fail(fmt.Sprintf(format, args...))
}

func (r *recorder) Fail() {
	r.Helper()
	r.failed = true
}

func (r *recorder) FailNow() {
	r.Helper()
	r.failed = true
	panic(RecorderFailNow)
}

func (r *recorder) Failed() bool {
	return r.failed
}

func (r *recorder) Fatal(args ...any) {
	r.Helper()
	r.failed = true
	r.fatal(fmt.Sprint(args...))
	panic(RecorderFatal)
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.Helper()
	r.failed = true
	r.fatal(fmt.Sprintf(format, args...))
	panic(RecorderFatalf)
}

func (r *recorder) Helper() {
	r.TB.Helper()
}
