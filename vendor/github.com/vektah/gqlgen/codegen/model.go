package codegen

type Model struct {
	*NamedType

	Fields []ModelField
}

type ModelField struct {
	*Type
	GQLName   string
	GoVarName string
	GoFKName  string
	GoFKType  string
}
