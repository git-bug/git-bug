package codegen

import (
	"go/types"
	"sort"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildInputs(namedTypes NamedTypes, prog *loader.Program) (Objects, error) {
	var inputs Objects

	for _, typ := range cfg.schema.Types {
		switch typ.Kind {
		case ast.InputObject:
			input, err := cfg.buildInput(namedTypes, typ)
			if err != nil {
				return nil, err
			}

			def, err := findGoType(prog, input.Package, input.GoType)
			if err != nil {
				return nil, errors.Wrap(err, "cannot find type")
			}
			if def != nil {
				input.Marshaler = buildInputMarshaler(typ, def)
				bindErrs := bindObject(def.Type(), input, cfg.StructTag)
				if len(bindErrs) > 0 {
					return nil, bindErrs
				}
			}

			inputs = append(inputs, input)
		}
	}

	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].GQLType < inputs[j].GQLType
	})

	return inputs, nil
}

func (cfg *Config) buildInput(types NamedTypes, typ *ast.Definition) (*Object, error) {
	obj := &Object{NamedType: types[typ.Name]}
	typeEntry, entryExists := cfg.Models[typ.Name]

	for _, field := range typ.Fields {
		newField := Field{
			GQLName: field.Name,
			Type:    types.getType(field.Type),
			Object:  obj,
		}

		if entryExists {
			if typeField, ok := typeEntry.Fields[field.Name]; ok {
				newField.GoFieldName = typeField.FieldName
			}
		}

		if field.DefaultValue != nil {
			var err error
			newField.Default, err = field.DefaultValue.Value(nil)
			if err != nil {
				return nil, errors.Errorf("default value for %s.%s is not valid: %s", typ.Name, field.Name, err.Error())
			}
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
func buildInputMarshaler(typ *ast.Definition, def types.Object) *Ref {
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
