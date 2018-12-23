package codegen

import (
	"sort"

	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildModels(types NamedTypes, prog *loader.Program) ([]Model, error) {
	var models []Model

	for _, typ := range cfg.schema.Types {
		var model Model
		switch typ.Kind {
		case ast.Object:
			obj, err := cfg.buildObject(types, typ)
			if err != nil {
				return nil, err
			}
			if obj.Root || obj.IsUserDefined {
				continue
			}
			model = cfg.obj2Model(obj)
		case ast.InputObject:
			obj, err := cfg.buildInput(types, typ)
			if err != nil {
				return nil, err
			}
			if obj.IsUserDefined {
				continue
			}
			model = cfg.obj2Model(obj)
		case ast.Interface, ast.Union:
			intf := cfg.buildInterface(types, typ, prog)
			if intf.IsUserDefined {
				continue
			}
			model = int2Model(intf)
		default:
			continue
		}
		model.Description = typ.Description // It's this or change both obj2Model and buildObject

		models = append(models, model)
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].GQLType < models[j].GQLType
	})

	return models, nil
}

func (cfg *Config) obj2Model(obj *Object) Model {
	model := Model{
		NamedType:  obj.NamedType,
		Implements: obj.Implements,
		Fields:     []ModelField{},
	}

	model.GoType = ucFirst(obj.GQLType)
	model.Marshaler = &Ref{GoType: obj.GoType}

	for i := range obj.Fields {
		field := &obj.Fields[i]
		mf := ModelField{Type: field.Type, GQLName: field.GQLName}

		if field.GoFieldName != "" {
			mf.GoFieldName = field.GoFieldName
		} else {
			mf.GoFieldName = field.GoNameExported()
		}

		model.Fields = append(model.Fields, mf)
	}

	return model
}

func int2Model(obj *Interface) Model {
	model := Model{
		NamedType: obj.NamedType,
		Fields:    []ModelField{},
	}

	model.GoType = ucFirst(obj.GQLType)
	model.Marshaler = &Ref{GoType: obj.GoType}

	return model
}
