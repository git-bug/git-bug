package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

type Directive struct {
	Name string
	Args []FieldArgument
}

func (d *Directive) ArgsFunc() string {
	if len(d.Args) == 0 {
		return ""
	}

	return "dir_" + d.Name + "_args"
}

func (d *Directive) CallArgs() string {
	args := []string{"ctx", "obj", "n"}

	for _, arg := range d.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+arg.Signature()+")")
	}

	return strings.Join(args, ", ")
}

func (d *Directive) Declaration() string {
	res := ucFirst(d.Name) + " func(ctx context.Context, obj interface{}, next graphql.Resolver"

	for _, arg := range d.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
	}

	res += ") (res interface{}, err error)"
	return res
}
