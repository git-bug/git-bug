package codegen

type Model struct {
	*NamedType
	Description string
	Fields      []ModelField
	Implements  []*NamedType
}

type ModelField struct {
	*Type
	GQLName     string
	GoFieldName string
	GoFKName    string
	GoFKType    string
	Description string
}
