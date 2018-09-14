package codegen

import (
	"sort"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

func (cfg *Config) buildEnums(types NamedTypes) []Enum {
	var enums []Enum

	for _, typ := range cfg.schema.Types {
		namedType := types[typ.Name]
		if typ.Kind != ast.Enum || strings.HasPrefix(typ.Name, "__") || namedType.IsUserDefined {
			continue
		}

		var values []EnumValue
		for _, v := range typ.EnumValues {
			values = append(values, EnumValue{v.Name, v.Description})
		}

		enum := Enum{
			NamedType:   namedType,
			Values:      values,
			Description: typ.Description,
		}
		enum.GoType = templates.ToCamel(enum.GQLType)
		enums = append(enums, enum)
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].GQLType < enums[j].GQLType
	})

	return enums
}
