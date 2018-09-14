package codegen

import (
	"fmt"
	"go/types"
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

func findGoType(prog *loader.Program, pkgName string, typeName string) (types.Object, error) {
	if pkgName == "" {
		return nil, nil
	}
	fullName := typeName
	if pkgName != "" {
		fullName = pkgName + "." + typeName
	}

	pkgName, err := resolvePkg(pkgName)
	if err != nil {
		return nil, errors.Errorf("unable to resolve package for %s: %s\n", fullName, err.Error())
	}

	pkg := prog.Imported[pkgName]
	if pkg == nil {
		return nil, errors.Errorf("required package was not loaded: %s", fullName)
	}

	for astNode, def := range pkg.Defs {
		if astNode.Name != typeName || def.Parent() == nil || def.Parent() != pkg.Pkg.Scope() {
			continue
		}

		return def, nil
	}

	return nil, errors.Errorf("unable to find type %s\n", fullName)
}

func findGoNamedType(prog *loader.Program, pkgName string, typeName string) (*types.Named, error) {
	def, err := findGoType(prog, pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, nil
	}

	namedType, ok := def.Type().(*types.Named)
	if !ok {
		return nil, errors.Errorf("expected %s to be a named type, instead found %T\n", typeName, def.Type())
	}

	return namedType, nil
}

func findGoInterface(prog *loader.Program, pkgName string, typeName string) (*types.Interface, error) {
	namedType, err := findGoNamedType(prog, pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if namedType == nil {
		return nil, nil
	}

	underlying, ok := namedType.Underlying().(*types.Interface)
	if !ok {
		return nil, errors.Errorf("expected %s to be a named interface, instead found %s", typeName, namedType.String())
	}

	return underlying, nil
}

func findMethod(typ *types.Named, name string) *types.Func {
	for i := 0; i < typ.NumMethods(); i++ {
		method := typ.Method(i)
		if !method.Exported() {
			continue
		}

		if strings.EqualFold(method.Name(), name) {
			return method
		}
	}

	if s, ok := typ.Underlying().(*types.Struct); ok {
		for i := 0; i < s.NumFields(); i++ {
			field := s.Field(i)
			if !field.Anonymous() {
				continue
			}

			if named, ok := field.Type().(*types.Named); ok {
				if f := findMethod(named, name); f != nil {
					return f
				}
			}
		}
	}

	return nil
}

// findField attempts to match the name to a struct field with the following
// priorites:
// 1. If struct tag is passed then struct tag has highest priority
// 2. Field in an embedded struct
// 3. Actual Field name
func findField(typ *types.Struct, name, structTag string) (*types.Var, error) {
	var foundField *types.Var
	foundFieldWasTag := false

	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)

		if structTag != "" {
			tags := reflect.StructTag(typ.Tag(i))
			if val, ok := tags.Lookup(structTag); ok {
				if strings.EqualFold(val, name) {
					if foundField != nil && foundFieldWasTag {
						return nil, errors.Errorf("tag %s is ambigious; multiple fields have the same tag value of %s", structTag, val)
					}

					foundField = field
					foundFieldWasTag = true
				}
			}
		}

		if field.Anonymous() {
			if named, ok := field.Type().(*types.Struct); ok {
				f, err := findField(named, name, structTag)
				if err != nil && !strings.HasPrefix(err.Error(), "no field named") {
					return nil, err
				}
				if f != nil && foundField == nil {
					foundField = f
				}
			}

			if named, ok := field.Type().Underlying().(*types.Struct); ok {
				f, err := findField(named, name, structTag)
				if err != nil && !strings.HasPrefix(err.Error(), "no field named") {
					return nil, err
				}
				if f != nil && foundField == nil {
					foundField = f
				}
			}
		}

		if !field.Exported() {
			continue
		}

		if strings.EqualFold(field.Name(), name) && foundField == nil {
			foundField = field
		}
	}

	if foundField == nil {
		return nil, fmt.Errorf("no field named %s", name)
	}

	return foundField, nil
}

type BindError struct {
	object    *Object
	field     *Field
	typ       types.Type
	methodErr error
	varErr    error
}

func (b BindError) Error() string {
	return fmt.Sprintf(
		"Unable to bind %s.%s to %s\n  %s\n  %s",
		b.object.GQLType,
		b.field.GQLName,
		b.typ.String(),
		b.methodErr.Error(),
		b.varErr.Error(),
	)
}

type BindErrors []BindError

func (b BindErrors) Error() string {
	var errs []string
	for _, err := range b {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, "\n\n")
}

func bindObject(t types.Type, object *Object, imports *Imports, structTag string) BindErrors {
	var errs BindErrors
	for i := range object.Fields {
		field := &object.Fields[i]

		if field.ForceResolver {
			continue
		}

		// first try binding to a method
		methodErr := bindMethod(imports, t, field)
		if methodErr == nil {
			continue
		}

		// otherwise try binding to a var
		varErr := bindVar(imports, t, field, structTag)

		if varErr != nil {
			errs = append(errs, BindError{
				object:    object,
				typ:       t,
				field:     field,
				varErr:    varErr,
				methodErr: methodErr,
			})
		}
	}
	return errs
}

func bindMethod(imports *Imports, t types.Type, field *Field) error {
	namedType, ok := t.(*types.Named)
	if !ok {
		return fmt.Errorf("not a named type")
	}

	goName := field.GQLName
	if field.GoFieldName != "" {
		goName = field.GoFieldName
	}
	method := findMethod(namedType, goName)
	if method == nil {
		return fmt.Errorf("no method named %s", field.GQLName)
	}
	sig := method.Type().(*types.Signature)

	if sig.Results().Len() == 1 {
		field.NoErr = true
	} else if sig.Results().Len() != 2 {
		return fmt.Errorf("method has wrong number of args")
	}
	newArgs, err := matchArgs(field, sig.Params())
	if err != nil {
		return err
	}

	result := sig.Results().At(0)
	if err := validateTypeBinding(imports, field, result.Type()); err != nil {
		return errors.Wrap(err, "method has wrong return type")
	}

	// success, args and return type match. Bind to method
	field.GoFieldType = GoFieldMethod
	field.GoReceiverName = "obj"
	field.GoFieldName = method.Name()
	field.Args = newArgs
	return nil
}

func bindVar(imports *Imports, t types.Type, field *Field, structTag string) error {
	underlying, ok := t.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("not a struct")
	}

	goName := field.GQLName
	if field.GoFieldName != "" {
		goName = field.GoFieldName
	}
	structField, err := findField(underlying, goName, structTag)
	if err != nil {
		return err
	}

	if err := validateTypeBinding(imports, field, structField.Type()); err != nil {
		return errors.Wrap(err, "field has wrong type")
	}

	// success, bind to var
	field.GoFieldType = GoFieldVariable
	field.GoReceiverName = "obj"
	field.GoFieldName = structField.Name()
	return nil
}

func matchArgs(field *Field, params *types.Tuple) ([]FieldArgument, error) {
	var newArgs []FieldArgument

nextArg:
	for j := 0; j < params.Len(); j++ {
		param := params.At(j)
		for _, oldArg := range field.Args {
			if strings.EqualFold(oldArg.GQLName, param.Name()) {
				if !field.ForceResolver {
					oldArg.Type.Modifiers = modifiersFromGoType(param.Type())
				}
				newArgs = append(newArgs, oldArg)
				continue nextArg
			}
		}

		// no matching arg found, abort
		return nil, fmt.Errorf("arg %s not found on method", param.Name())
	}
	return newArgs, nil
}

func validateTypeBinding(imports *Imports, field *Field, goType types.Type) error {
	gqlType := normalizeVendor(field.Type.FullSignature())
	goTypeStr := normalizeVendor(goType.String())

	if goTypeStr == gqlType || "*"+goTypeStr == gqlType || goTypeStr == "*"+gqlType {
		field.Type.Modifiers = modifiersFromGoType(goType)
		return nil
	}

	// deal with type aliases
	underlyingStr := normalizeVendor(goType.Underlying().String())
	if underlyingStr == gqlType || "*"+underlyingStr == gqlType || underlyingStr == "*"+gqlType {
		field.Type.Modifiers = modifiersFromGoType(goType)
		pkg, typ := pkgAndType(goType.String())
		imp := imports.findByPath(pkg)
		field.AliasedType = &Ref{GoType: typ, Import: imp}
		return nil
	}

	return fmt.Errorf("%s is not compatible with %s", gqlType, goTypeStr)
}

func modifiersFromGoType(t types.Type) []string {
	var modifiers []string
	for {
		switch val := t.(type) {
		case *types.Pointer:
			modifiers = append(modifiers, modPtr)
			t = val.Elem()
		case *types.Array:
			modifiers = append(modifiers, modList)
			t = val.Elem()
		case *types.Slice:
			modifiers = append(modifiers, modList)
			t = val.Elem()
		default:
			return modifiers
		}
	}
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

func normalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}
