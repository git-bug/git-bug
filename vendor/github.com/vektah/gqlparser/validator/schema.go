//go:generate go run ./inliner/inliner.go

package validator

import (
	"strconv"

	. "github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
	"github.com/vektah/gqlparser/parser"
)

func LoadSchema(inputs ...*Source) (*Schema, *gqlerror.Error) {
	ast := &SchemaDocument{}
	for _, input := range inputs {
		inputAst, err := parser.ParseSchema(input)
		if err != nil {
			return nil, err
		}

		ast.Merge(inputAst)
	}

	schema := Schema{
		Types:         map[string]*Definition{},
		Directives:    map[string]*DirectiveDefinition{},
		PossibleTypes: map[string][]*Definition{},
	}

	for i, def := range ast.Definitions {
		if schema.Types[def.Name] != nil {
			return nil, gqlerror.ErrorPosf(def.Position, "Cannot redeclare type %s.", def.Name)
		}
		schema.Types[def.Name] = ast.Definitions[i]

		if def.Kind != Interface {
			for _, intf := range def.Interfaces {
				schema.AddPossibleType(intf, ast.Definitions[i])
			}
			schema.AddPossibleType(def.Name, ast.Definitions[i])
		}
	}

	for _, ext := range ast.Extensions {
		def := schema.Types[ext.Name]
		if def == nil {
			return nil, gqlerror.ErrorPosf(ext.Position, "Cannot extend type %s because it does not exist.", ext.Name)
		}

		if def.Kind != ext.Kind {
			return nil, gqlerror.ErrorPosf(ext.Position, "Cannot extend type %s because the base type is a %s, not %s.", ext.Name, def.Kind, ext.Kind)
		}

		def.Directives = append(def.Directives, ext.Directives...)
		def.Interfaces = append(def.Interfaces, ext.Interfaces...)
		def.Fields = append(def.Fields, ext.Fields...)
		def.Types = append(def.Types, ext.Types...)
		def.EnumValues = append(def.EnumValues, ext.EnumValues...)
	}

	for i, dir := range ast.Directives {
		if schema.Directives[dir.Name] != nil {
			return nil, gqlerror.ErrorPosf(dir.Position, "Cannot redeclare directive %s.", dir.Name)
		}
		schema.Directives[dir.Name] = ast.Directives[i]
	}

	if len(ast.Schema) > 1 {
		return nil, gqlerror.ErrorPosf(ast.Schema[1].Position, "Cannot have multiple schema entry points, consider schema extensions instead.")
	}

	if len(ast.Schema) == 1 {
		for _, entrypoint := range ast.Schema[0].OperationTypes {
			def := schema.Types[entrypoint.Type]
			if def == nil {
				return nil, gqlerror.ErrorPosf(entrypoint.Position, "Schema root %s refers to a type %s that does not exist.", entrypoint.Operation, entrypoint.Type)
			}
			switch entrypoint.Operation {
			case Query:
				schema.Query = def
			case Mutation:
				schema.Mutation = def
			case Subscription:
				schema.Subscription = def
			}
		}
	}

	for _, ext := range ast.SchemaExtension {
		for _, entrypoint := range ext.OperationTypes {
			def := schema.Types[entrypoint.Type]
			if def == nil {
				return nil, gqlerror.ErrorPosf(entrypoint.Position, "Schema root %s refers to a type %s that does not exist.", entrypoint.Operation, entrypoint.Type)
			}
			switch entrypoint.Operation {
			case Query:
				schema.Query = def
			case Mutation:
				schema.Mutation = def
			case Subscription:
				schema.Subscription = def
			}
		}
	}

	for _, typ := range schema.Types {
		err := validateDefinition(&schema, typ)
		if err != nil {
			return nil, err
		}
	}

	for _, dir := range schema.Directives {
		err := validateDirective(&schema, dir)
		if err != nil {
			return nil, err
		}
	}

	if schema.Query == nil && schema.Types["Query"] != nil {
		schema.Query = schema.Types["Query"]
	}

	if schema.Mutation == nil && schema.Types["Mutation"] != nil {
		schema.Mutation = schema.Types["Mutation"]
	}

	if schema.Subscription == nil && schema.Types["Subscription"] != nil {
		schema.Subscription = schema.Types["Subscription"]
	}

	if schema.Query != nil {
		schema.Query.Fields = append(
			schema.Query.Fields,
			&FieldDefinition{
				Name: "__schema",
				Type: NonNullNamedType("__Schema", nil),
			},
			&FieldDefinition{
				Name: "__type",
				Type: NonNullNamedType("__Type", nil),
				Arguments: ArgumentDefinitionList{
					{Name: "name", Type: NamedType("String", nil)},
				},
			},
		)
	}

	return &schema, nil
}

func validateDirective(schema *Schema, def *DirectiveDefinition) *gqlerror.Error {
	return validateArgs(schema, def.Arguments, def)
}

func validateDefinition(schema *Schema, def *Definition) *gqlerror.Error {
	for _, field := range def.Fields {
		if err := validateTypeRef(schema, field.Type); err != nil {
			return err
		}
		if err := validateArgs(schema, field.Arguments, nil); err != nil {
			return err
		}
		if err := validateDirectives(schema, field.Directives, nil); err != nil {
			return err
		}
	}

	for _, intf := range def.Interfaces {
		intDef := schema.Types[intf]
		if intDef == nil {
			return gqlerror.ErrorPosf(def.Position, "Undefined type %s.", strconv.Quote(intf))
		}
		if intDef.Kind != Interface {
			return gqlerror.ErrorPosf(def.Position, "%s is a non interface type %s.", strconv.Quote(intf), intDef.Kind)
		}
	}

	return validateDirectives(schema, def.Directives, nil)
}

func validateTypeRef(schema *Schema, typ *Type) *gqlerror.Error {
	if schema.Types[typ.Name()] == nil {
		return gqlerror.ErrorPosf(typ.Position, "Undefined type %s.", typ.Name())
	}
	return nil
}

func validateArgs(schema *Schema, args ArgumentDefinitionList, currentDirective *DirectiveDefinition) *gqlerror.Error {
	for _, arg := range args {
		if err := validateTypeRef(schema, arg.Type); err != nil {
			return err
		}
		if err := validateDirectives(schema, arg.Directives, currentDirective); err != nil {
			return err
		}
	}
	return nil
}

func validateDirectives(schema *Schema, dirs DirectiveList, currentDirective *DirectiveDefinition) *gqlerror.Error {
	for _, dir := range dirs {
		if currentDirective != nil && dir.Name == currentDirective.Name {
			return gqlerror.ErrorPosf(dir.Position, "Directive %s cannot refer to itself.", currentDirective.Name)
		}
		if schema.Directives[dir.Name] == nil {
			return gqlerror.ErrorPosf(dir.Position, "Undefined directive %s.", dir.Name)
		}
		dir.Definition = schema.Directives[dir.Name]
	}
	return nil
}
