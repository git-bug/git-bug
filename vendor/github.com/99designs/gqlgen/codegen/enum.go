package codegen

type Enum struct {
	*NamedType
	Description string
	Values      []EnumValue
}

type EnumValue struct {
	Name        string
	Description string
}
