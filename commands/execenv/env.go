package execenv

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"golang.org/x/term"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

const RootCommandName = "git-bug"

const gitBugNamespace = "git-bug"

// Env is the environment of a command
type Env struct {
	Repo     repository.ClockedRepo
	Backend  *cache.RepoCache
	In       In
	Out      Out
	Err      Out
	RepoPath []string
}

func NewEnv() *Env {
	return &Env{
		Repo: nil,
		In:   in{Reader: os.Stdin},
		Out:  out{Writer: os.Stdout},
		Err:  out{Writer: os.Stderr},
	}
}

type In interface {
	io.Reader

	// IsTerminal tells if the input is a user terminal (rather than a buffer,
	// a pipe ...), which tells if we can use interactive features.
	IsTerminal() bool

	// ForceIsTerminal allow to force the returned value of IsTerminal
	// This only works in test scenario.
	ForceIsTerminal(value bool)
}

type Out interface {
	io.Writer

	Printf(format string, a ...interface{})
	Print(a ...interface{})
	Println(a ...interface{})
	PrintJSON(v interface{}) error

	// IsTerminal tells if the output is a user terminal (rather than a buffer,
	// a pipe ...), which tells if we can use colors and other interactive features.
	IsTerminal() bool
	// Width return the width of the attached terminal, or a good enough value.
	Width() int

	// Raw return the underlying io.Writer, or itself if not.
	// This is useful if something need to access the raw file descriptor.
	Raw() io.Writer

	// String returns what have been written in the output before, as a string.
	// This only works in test scenario.
	String() string
	// Bytes returns what have been written in the output before, as []byte.
	// This only works in test scenario.
	Bytes() []byte
	// Reset clear what has been recorded as written in the output before.
	// This only works in test scenario.
	Reset()
	// ForceIsTerminal allow to force the returned value of IsTerminal
	// This only works in test scenario.
	ForceIsTerminal(value bool)
}

type in struct {
	io.Reader
}

func (i in) IsTerminal() bool {
	if f, ok := i.Reader.(*os.File); ok {
		return isTerminal(f)
	}
	return false
}

func (i in) ForceIsTerminal(_ bool) {
	panic("only work with a test env")
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

func (o out) PrintJSON(v interface{}) error {
	raw, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	o.Println(string(raw))
	return nil
}

func (o out) IsTerminal() bool {
	if f, ok := o.Writer.(*os.File); ok {
		return isTerminal(f)
	}
	return false
}

func (o out) Width() int {
	if f, ok := o.Raw().(*os.File); ok {
		width, _, err := term.GetSize(int(f.Fd()))
		if err == nil {
			return width
		}
	}
	return 80
}

func (o out) Raw() io.Writer {
	return o.Writer
}

func (o out) String() string {
	panic("only work with a test env")
}

func (o out) Bytes() []byte {
	panic("only work with a test env")
}

func (o out) Reset() {
	panic("only work with a test env")
}

func (o out) ForceIsTerminal(_ bool) {
	panic("only work with a test env")
}

func isTerminal(file *os.File) bool {
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}
