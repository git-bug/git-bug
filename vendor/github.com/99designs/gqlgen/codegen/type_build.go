package codegen

import (
	"go/types"
	"strings"

	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

// namedTypeFromSchema objects for every graphql type, including scalars. There should only be one instance of Type for each thing
func (cfg *Config) buildNamedTypes() NamedTypes {
	types := map[string]*NamedType{}
	for _, schemaType := range cfg.schema.Types {
		t := namedTypeFromSchema(schemaType)

		if userEntry, ok := cfg.Models[t.GQLType]; ok && userEntry.Model != "" {
			t.IsUserDefined = true
			t.Package, t.GoType = pkgAndType(userEntry.Model)
		} else if t.IsScalar {
			t.Package = "github.com/99designs/gqlgen/graphql"
			t.GoType = "String"
		}

		types[t.GQLType] = t
	}
	return types
}

func (cfg *Config) bindTypes(namedTypes NamedTypes, destDir string, prog *loader.Program) {
	for _, t := range namedTypes {
		if t.Package == "" {
			continue
		}

		def, _ := findGoType(prog, t.Package, "Marshal"+t.GoType)
		switch def := def.(type) {
		case *types.Func:
			sig := def.Type().(*types.Signature)
			cpy := t.Ref
			t.Marshaler = &cpy

			t.Package, t.GoType = pkgAndType(sig.Params().At(0).Type().String())
		}
	}
}

// namedTypeFromSchema objects for every graphql type, including primitives.
// don't recurse into object fields or interfaces yet, lets make sure we have collected everything first.
func namedTypeFromSchema(schemaType *ast.Definition) *NamedType {
	switch schemaType.Kind {
	case ast.Scalar, ast.Enum:
		return &NamedType{GQLType: schemaType.Name, IsScalar: true}
	case ast.Interface, ast.Union:
		return &NamedType{GQLType: schemaType.Name, IsInterface: true}
	case ast.InputObject:
		return &NamedType{GQLType: schemaType.Name, IsInput: true}
	default:
		return &NamedType{GQLType: schemaType.Name}
	}
}

// take a string in the form github.com/package/blah.Type and split it into package and type
func pkgAndType(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", name
	}

	return normalizeVendor(strings.Join(parts[:len(parts)-1], ".")), parts[len(parts)-1]
}

func (n NamedTypes) getType(t *ast.Type) *Type {
	orig := t
	var modifiers []string
	for {
		if t.Elem != nil {
			modifiers = append(modifiers, modList)
			t = t.Elem
		} else {
			if !t.NonNull {
				modifiers = append(modifiers, modPtr)
			}
			if n[t.NamedType] == nil {
				panic("missing type " + t.NamedType)
			}
			res := &Type{
				NamedType: n[t.NamedType],
				Modifiers: modifiers,
				ASTType:   orig,
			}

			if res.IsInterface {
				res.StripPtr()
			}

			return res
		}
	}
}
