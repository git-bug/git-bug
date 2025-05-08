package test

import (
	"fmt"
	"testing"
)

type recorder struct {
	testing.TB
	fail  func(string)
	fatal func(string)
}

func (r *recorder) Errorf(format string, args ...any) {
	r.fail(fmt.Sprintf(format, args...))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.fatal(fmt.Sprintf(format, args...))
}

func (r *recorder) Fatal(args ...any) {
	r.fatal(fmt.Sprint(args...))
}

func (r *recorder) Error(args ...any) {
	r.fail(fmt.Sprint(args...))
}
