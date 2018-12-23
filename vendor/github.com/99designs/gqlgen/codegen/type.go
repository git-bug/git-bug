package codegen

import (
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"

	"github.com/vektah/gqlparser/ast"
)

type NamedTypes map[string]*NamedType

type NamedType struct {
	Ref
	IsScalar    bool
	IsInterface bool
	IsInput     bool
	GQLType     string // Name of the graphql type
	Marshaler   *Ref   // If this type has an external marshaler this will be set
}

type Ref struct {
	GoType        string // Name of the go type
	Package       string // the package the go type lives in
	IsUserDefined bool   // does the type exist in the typemap
}

type Type struct {
	*NamedType

	Modifiers   []string
	ASTType     *ast.Type
	AliasedType *Ref
}

const (
	modList = "[]"
	modPtr  = "*"
)

func (t Ref) FullName() string {
	return t.PkgDot() + t.GoType
}

func (t Ref) PkgDot() string {
	name := templates.CurrentImports.Lookup(t.Package)
	if name == "" {
		return ""

	}

	return name + "."
}

func (t Type) Signature() string {
	if t.AliasedType != nil {
		return strings.Join(t.Modifiers, "") + t.AliasedType.FullName()
	}
	return strings.Join(t.Modifiers, "") + t.FullName()
}

func (t Type) FullSignature() string {
	pkg := ""
	if t.Package != "" {
		pkg = t.Package + "."
	}

	return strings.Join(t.Modifiers, "") + pkg + t.GoType
}

func (t Type) IsPtr() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modPtr
}

func (t *Type) StripPtr() {
	if !t.IsPtr() {
		return
	}
	t.Modifiers = t.Modifiers[0 : len(t.Modifiers)-1]
}

func (t Type) IsSlice() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modList ||
		len(t.Modifiers) > 1 && t.Modifiers[0] == modPtr && t.Modifiers[1] == modList
}

func (t NamedType) IsMarshaled() bool {
	return t.Marshaler != nil
}

func (t Type) Unmarshal(result, raw string) string {
	return t.unmarshal(result, raw, t.Modifiers, 1)
}

func (t Type) unmarshal(result, raw string, remainingMods []string, depth int) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		ptr := "ptr" + strconv.Itoa(depth)
		return tpl(`var {{.ptr}} {{.mods}}{{.t.FullName}}
			if {{.raw}} != nil {
				{{.next}}
				{{.result}} = &{{.ptr -}}
			}
		`, map[string]interface{}{
			"ptr":    ptr,
			"t":      t,
			"raw":    raw,
			"result": result,
			"mods":   strings.Join(remainingMods[1:], ""),
			"next":   t.unmarshal(ptr, raw, remainingMods[1:], depth+1),
		})

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		var rawIf = "rawIf" + strconv.Itoa(depth)
		var index = "idx" + strconv.Itoa(depth)

		return tpl(`var {{.rawSlice}} []interface{}
			if {{.raw}} != nil {
				if tmp1, ok := {{.raw}}.([]interface{}); ok {
					{{.rawSlice}} = tmp1
				} else {
					{{.rawSlice}} = []interface{}{ {{.raw}} }
				}
			}
			{{.result}} = make({{.type}}, len({{.rawSlice}}))
			for {{.index}} := range {{.rawSlice}} {
				{{ .next -}}
			}`, map[string]interface{}{
			"raw":      raw,
			"rawSlice": rawIf,
			"index":    index,
			"result":   result,
			"type":     strings.Join(remainingMods, "") + t.NamedType.FullName(),
			"next":     t.unmarshal(result+"["+index+"]", rawIf+"["+index+"]", remainingMods[1:], depth+1),
		})
	}

	realResult := result
	if t.AliasedType != nil {
		result = "castTmp"
	}

	return tpl(`{{- if .t.AliasedType }}
			var castTmp {{.t.FullName}}
		{{ end }}
			{{- if eq .t.GoType "map[string]interface{}" }}
				{{- .result }} = {{.raw}}.(map[string]interface{})
			{{- else if .t.Marshaler }}
				{{- .result }}, err = {{ .t.Marshaler.PkgDot }}Unmarshal{{.t.Marshaler.GoType}}({{.raw}})
			{{- else -}}
				err = (&{{.result}}).UnmarshalGQL({{.raw}})
			{{- end }}
		{{- if .t.AliasedType }}
			{{ .realResult }} = {{.t.AliasedType.FullName}}(castTmp)
		{{- end }}`, map[string]interface{}{
		"realResult": realResult,
		"result":     result,
		"raw":        raw,
		"t":          t,
	})
}

func (t Type) Marshal(val string) string {
	if t.AliasedType != nil {
		val = t.GoType + "(" + val + ")"
	}

	if t.Marshaler != nil {
		return "return " + t.Marshaler.PkgDot() + "Marshal" + t.Marshaler.GoType + "(" + val + ")"
	}

	return "return " + val
}
