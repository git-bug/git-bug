package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/MichaelMure/git-bug/repository"
)

// Env is the environment of a command
type Env struct {
	repo repository.ClockedRepo
	out  out
	err  out
}

func newEnv() *Env {
	return &Env{
		repo: nil,
		out:  out{Writer: os.Stdout},
		err:  out{Writer: os.Stderr},
	}
}

type out struct {
	io.Writer
}

func (o out) Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(o, format, a...)
}

func (o out) Print(a ...interface{}) {
	_, _ = fmt.Fprint(o, a...)
}

func (o out) Println(a ...interface{}) {
	_, _ = fmt.Fprintln(o, a...)
}
