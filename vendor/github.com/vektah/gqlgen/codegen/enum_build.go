package codegen

import (
	"sort"
	"strings"

	"github.com/vektah/gqlgen/codegen/templates"
	"github.com/vektah/gqlgen/neelance/schema"
)

func (cfg *Config) buildEnums(types NamedTypes) []Enum {
	var enums []Enum

	for _, typ := range cfg.schema.Types {
		namedType := types[typ.TypeName()]
		e, isEnum := typ.(*schema.Enum)
		if !isEnum || strings.HasPrefix(typ.TypeName(), "__") || namedType.IsUserDefined {
			continue
		}

		var values []EnumValue
		for _, v := range e.Values {
			values = append(values, EnumValue{v.Name, v.Desc})
		}

		enum := Enum{
			NamedType: namedType,
			Values:    values,
		}
		enum.GoType = templates.ToCamel(enum.GQLType)
		enums = append(enums, enum)
	}

	sort.Slice(enums, func(i, j int) bool {
		return strings.Compare(enums[i].GQLType, enums[j].GQLType) == -1
	})

	return enums
}
