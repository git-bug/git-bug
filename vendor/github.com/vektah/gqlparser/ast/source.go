package ast

type Source struct {
	Name    string
	Input   string
	BuiltIn bool
}

type Position struct {
	Start  int     // The starting position, in runes, of this token in the input.
	End    int     // The end position, in runes, of this token in the input.
	Line   int     // The line number at the start of this item.
	Column int     // The line number at the start of this item.
	Src    *Source // The source document this token belongs to
}
