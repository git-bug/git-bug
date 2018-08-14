package codegen

type Enum struct {
	*NamedType

	Values []EnumValue
}

type EnumValue struct {
	Name        string
	Description string
}
