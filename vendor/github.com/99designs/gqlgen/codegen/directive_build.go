package codegen

import (
	"sort"

	"github.com/pkg/errors"
)

func (cfg *Config) buildDirectives(types NamedTypes) ([]*Directive, error) {
	var directives []*Directive

	for name, dir := range cfg.schema.Directives {
		if name == "skip" || name == "include" || name == "deprecated" {
			continue
		}

		var args []FieldArgument
		for _, arg := range dir.Arguments {
			newArg := FieldArgument{
				GQLName:   arg.Name,
				Type:      types.getType(arg.Type),
				GoVarName: sanitizeArgName(arg.Name),
			}

			if !newArg.Type.IsInput && !newArg.Type.IsScalar {
				return nil, errors.Errorf("%s cannot be used as argument of directive %s(%s) only input and scalar types are allowed", arg.Type, dir.Name, arg.Name)
			}

			if arg.DefaultValue != nil {
				var err error
				newArg.Default, err = arg.DefaultValue.Value(nil)
				if err != nil {
					return nil, errors.Errorf("default value for directive argument %s(%s) is not valid: %s", dir.Name, arg.Name, err.Error())
				}
			}
			args = append(args, newArg)
		}

		directives = append(directives, &Directive{
			Name: name,
			Args: args,
		})
	}

	sort.Slice(directives, func(i, j int) bool { return directives[i].Name < directives[j].Name })

	return directives, nil
}
