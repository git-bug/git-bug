package codegen

import (
	"go/types"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildInputs(namedTypes NamedTypes, prog *loader.Program, imports *Imports) (Objects, error) {
	var inputs Objects

	for _, typ := range cfg.schema.Types {
		switch typ := typ.(type) {
		case *schema.InputObject:
			input, err := buildInput(namedTypes, typ)
			if err != nil {
				return nil, err
			}

			def, err := findGoType(prog, input.Package, input.GoType)
			if err != nil {
				return nil, errors.Wrap(err, "cannot find type")
			}
			if def != nil {
				input.Marshaler = buildInputMarshaler(typ, def)
				bindErrs := bindObject(def.Type(), input, imports)
				if len(bindErrs) > 0 {
					return nil, bindErrs
				}
			}

			inputs = append(inputs, input)
		}
	}

	sort.Slice(inputs, func(i, j int) bool {
		return strings.Compare(inputs[i].GQLType, inputs[j].GQLType) == -1
	})

	return inputs, nil
}

func buildInput(types NamedTypes, typ *schema.InputObject) (*Object, error) {
	obj := &Object{NamedType: types[typ.TypeName()]}

	for _, field := range typ.Values {
		newField := Field{
			GQLName: field.Name.Name,
			Type:    types.getType(field.Type),
			Object:  obj,
		}

		if field.Default != nil {
			newField.Default = field.Default.Value(nil)
		}

		if !newField.Type.IsInput && !newField.Type.IsScalar {
			return nil, errors.Errorf("%s cannot be used as a field of %s. only input and scalar types are allowed", newField.GQLType, obj.GQLType)
		}

		obj.Fields = append(obj.Fields, newField)

	}
	return obj, nil
}

// if user has implemented an UnmarshalGQL method on the input type manually, use it
// otherwise we will generate one.
func buildInputMarshaler(typ *schema.InputObject, def types.Object) *Ref {
	switch def := def.(type) {
	case *types.TypeName:
		namedType := def.Type().(*types.Named)
		for i := 0; i < namedType.NumMethods(); i++ {
			method := namedType.Method(i)
			if method.Name() == "UnmarshalGQL" {
				return nil
			}
		}
	}

	return &Ref{GoType: typ.Name}
}
