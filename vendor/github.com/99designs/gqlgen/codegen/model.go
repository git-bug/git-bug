package codegen

type Model struct {
	*NamedType
	Description string
	Fields      []ModelField
}

type ModelField struct {
	*Type
	GQLName     string
	GoFieldName string
	GoFKName    string
	GoFKType    string
	Description string
}
