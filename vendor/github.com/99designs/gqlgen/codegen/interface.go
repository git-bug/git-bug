package codegen

type Interface struct {
	*NamedType

	Implementors []InterfaceImplementor
}

type InterfaceImplementor struct {
	ValueReceiver bool

	*NamedType
}
