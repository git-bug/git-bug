package codegen

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/vektah/gqlparser/ast"
)

type GoFieldType int

const (
	GoFieldUndefined GoFieldType = iota
	GoFieldMethod
	GoFieldVariable
)

type Object struct {
	*NamedType

	Fields             []Field
	Satisfies          []string
	Implements         []*NamedType
	ResolverInterface  *Ref
	Root               bool
	DisableConcurrency bool
	Stream             bool
}

type Field struct {
	*Type
	Description      string          // Description of a field
	GQLName          string          // The name of the field in graphql
	GoFieldType      GoFieldType     // The field type in go, if any
	GoReceiverName   string          // The name of method & var receiver in go, if any
	GoFieldName      string          // The name of the method or var in go, if any
	Args             []FieldArgument // A list of arguments to be passed to this field
	ForceResolver    bool            // Should be emit Resolver method
	MethodHasContext bool            // If this is bound to a go method, does the method also take a context
	NoErr            bool            // If this is bound to a go method, does that method have an error as the second argument
	Object           *Object         // A link back to the parent object
	Default          interface{}     // The default value
}

type FieldArgument struct {
	*Type

	GQLName   string      // The name of the argument in graphql
	GoVarName string      // The name of the var in go
	Object    *Object     // A link back to the parent object
	Default   interface{} // The default value
}

type Objects []*Object

func (o *Object) Implementors() string {
	satisfiedBy := strconv.Quote(o.GQLType)
	for _, s := range o.Satisfies {
		satisfiedBy += ", " + strconv.Quote(s)
	}
	return "[]string{" + satisfiedBy + "}"
}

func (o *Object) HasResolvers() bool {
	for _, f := range o.Fields {
		if f.IsResolver() {
			return true
		}
	}
	return false
}

func (o *Object) IsConcurrent() bool {
	for _, f := range o.Fields {
		if f.IsConcurrent() {
			return true
		}
	}
	return false
}

func (o *Object) IsReserved() bool {
	return strings.HasPrefix(o.GQLType, "__")
}

func (f *Field) IsResolver() bool {
	return f.GoFieldName == ""
}

func (f *Field) IsReserved() bool {
	return strings.HasPrefix(f.GQLName, "__")
}

func (f *Field) IsMethod() bool {
	return f.GoFieldType == GoFieldMethod
}

func (f *Field) IsVariable() bool {
	return f.GoFieldType == GoFieldVariable
}

func (f *Field) IsConcurrent() bool {
	if f.Object.DisableConcurrency {
		return false
	}
	return f.MethodHasContext || f.IsResolver()
}

func (f *Field) GoNameExported() string {
	return lintName(ucFirst(f.GQLName))
}

func (f *Field) GoNameUnexported() string {
	return lintName(f.GQLName)
}

func (f *Field) ShortInvocation() string {
	if !f.IsResolver() {
		return ""
	}

	return fmt.Sprintf("%s().%s(%s)", f.Object.GQLType, f.GoNameExported(), f.CallArgs())
}

func (f *Field) ArgsFunc() string {
	if len(f.Args) == 0 {
		return ""
	}

	return "field_" + f.Object.GQLType + "_" + f.GQLName + "_args"
}

func (f *Field) ResolverType() string {
	if !f.IsResolver() {
		return ""
	}

	return fmt.Sprintf("%s().%s(%s)", f.Object.GQLType, f.GoNameExported(), f.CallArgs())
}

func (f *Field) ShortResolverDeclaration() string {
	if !f.IsResolver() {
		return ""
	}
	res := fmt.Sprintf("%s(ctx context.Context", f.GoNameExported())

	if !f.Object.Root {
		res += fmt.Sprintf(", obj *%s", f.Object.FullName())
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
	}

	result := f.Signature()
	if f.Object.Stream {
		result = "<-chan " + result
	}

	res += fmt.Sprintf(") (%s, error)", result)
	return res
}

func (f *Field) ResolverDeclaration() string {
	if !f.IsResolver() {
		return ""
	}
	res := fmt.Sprintf("%s_%s(ctx context.Context", f.Object.GQLType, f.GoNameUnexported())

	if !f.Object.Root {
		res += fmt.Sprintf(", obj *%s", f.Object.FullName())
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
	}

	result := f.Signature()
	if f.Object.Stream {
		result = "<-chan " + result
	}

	res += fmt.Sprintf(") (%s, error)", result)
	return res
}

func (f *Field) ComplexitySignature() string {
	res := fmt.Sprintf("func(childComplexity int")
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
	}
	res += ") int"
	return res
}

func (f *Field) ComplexityArgs() string {
	var args []string
	for _, arg := range f.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+arg.Signature()+")")
	}

	return strings.Join(args, ", ")
}

func (f *Field) CallArgs() string {
	var args []string

	if f.IsResolver() {
		args = append(args, "rctx")

		if !f.Object.Root {
			args = append(args, "obj")
		}
	} else {
		if f.MethodHasContext {
			args = append(args, "ctx")
		}
	}

	for _, arg := range f.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+arg.Signature()+")")
	}

	return strings.Join(args, ", ")
}

// should be in the template, but its recursive and has a bunch of args
func (f *Field) WriteJson() string {
	return f.doWriteJson("res", f.Type.Modifiers, f.ASTType, false, 1)
}

func (f *Field) doWriteJson(val string, remainingMods []string, astType *ast.Type, isPtr bool, depth int) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		return tpl(`
			if {{.val}} == nil {
				{{- if .nonNull }}
					if !ec.HasError(rctx) {
						ec.Errorf(ctx, "must not be null")
					}
				{{- end }}
				return graphql.Null
			}
			{{.next }}`, map[string]interface{}{
			"val":     val,
			"nonNull": astType.NonNull,
			"next":    f.doWriteJson(val, remainingMods[1:], astType, true, depth+1),
		})

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		if isPtr {
			val = "*" + val
		}
		var arr = "arr" + strconv.Itoa(depth)
		var index = "idx" + strconv.Itoa(depth)
		var usePtr bool
		if len(remainingMods) == 1 && !isPtr {
			usePtr = true
		}

		return tpl(`
			{{.arr}} := make(graphql.Array, len({{.val}}))
			{{ if and .top (not .isScalar) }} var wg sync.WaitGroup {{ end }}
			{{ if not .isScalar }}
				isLen1 := len({{.val}}) == 1
				if !isLen1 {
					wg.Add(len({{.val}}))
				}
			{{ end }}
			for {{.index}} := range {{.val}} {
				{{- if not .isScalar }}
					{{.index}} := {{.index}}
					rctx := &graphql.ResolverContext{
						Index: &{{.index}},
						Result: {{ if .usePtr }}&{{end}}{{.val}}[{{.index}}],
					}
					ctx := graphql.WithResolverContext(ctx, rctx)
					f := func({{.index}} int) {
						if !isLen1 {
							defer wg.Done()
						}
						{{.arr}}[{{.index}}] = func() graphql.Marshaler {
							{{ .next }}
						}()
					}
					if isLen1 {
						f({{.index}})
					} else {
						go f({{.index}})
					}
				{{ else }}
					{{.arr}}[{{.index}}] = func() graphql.Marshaler {
						{{ .next }}
					}()
				{{- end}}
			}
			{{ if and .top (not .isScalar) }} wg.Wait() {{ end }}
			return {{.arr}}`, map[string]interface{}{
			"val":      val,
			"arr":      arr,
			"index":    index,
			"top":      depth == 1,
			"arrayLen": len(val),
			"isScalar": f.IsScalar,
			"usePtr":   usePtr,
			"next":     f.doWriteJson(val+"["+index+"]", remainingMods[1:], astType.Elem, false, depth+1),
		})

	case f.IsScalar:
		if isPtr {
			val = "*" + val
		}
		return f.Marshal(val)

	default:
		if !isPtr {
			val = "&" + val
		}
		return tpl(`
			return ec._{{.type}}(ctx, field.Selections, {{.val}})`, map[string]interface{}{
			"type": f.GQLType,
			"val":  val,
		})
	}
}

func (f *FieldArgument) Stream() bool {
	return f.Object != nil && f.Object.Stream
}

func (os Objects) ByName(name string) *Object {
	for i, o := range os {
		if strings.EqualFold(o.GQLType, name) {
			return os[i]
		}
	}
	return nil
}

func tpl(tpl string, vars map[string]interface{}) string {
	b := &bytes.Buffer{}
	err := template.Must(template.New("inline").Parse(tpl)).Execute(b, vars)
	if err != nil {
		panic(err)
	}
	return b.String()
}

func ucFirst(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// copy from https://github.com/golang/lint/blob/06c8688daad7faa9da5a0c2f163a3d14aac986ca/lint.go#L679

// lintName returns a different name if it should be different.
func lintName(name string) (should string) {
	// Fast path for simple cases: "_" and all lowercase.
	if name == "_" {
		return name
	}
	allLower := true
	for _, r := range name {
		if !unicode.IsLower(r) {
			allLower = false
			break
		}
	}
	if allLower {
		return name
	}

	// Split camelCase at any lower->upper transition, and split on underscores.
	// Check each word for common initialisms.
	runes := []rune(name)
	w, i := 0, 0 // index of start of word, scan
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word
		if i+1 == len(runes) {
			eow = true
		} else if runes[i+1] == '_' {
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && runes[i+n+1] == '_' {
				n++
			}

			// Leave at most one underscore if the underscore is between two digits
			if i+n+1 < len(runes) && unicode.IsDigit(runes[i]) && unicode.IsDigit(runes[i+n+1]) {
				n--
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			// lower->non-lower
			eow = true
		}
		i++
		if !eow {
			continue
		}

		// [w,i) is a word.
		word := string(runes[w:i])
		if u := strings.ToUpper(word); commonInitialisms[u] {
			// Keep consistent case, which is lowercase only at the start.
			if w == 0 && unicode.IsLower(runes[w]) {
				u = strings.ToLower(u)
			}
			// All the common initialisms are ASCII,
			// so we can replace the bytes exactly.
			copy(runes[w:], []rune(u))
		} else if w > 0 && strings.ToLower(word) == word {
			// already all lowercase, and not the first word, so uppercase the first character.
			runes[w] = unicode.ToUpper(runes[w])
		}
		w = i
	}
	return string(runes)
}

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}
