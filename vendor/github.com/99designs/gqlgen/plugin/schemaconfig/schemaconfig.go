package schemaconfig

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/vektah/gqlparser/ast"
)

func New() plugin.Plugin {
	return &Plugin{}
}

type Plugin struct{}

var _ plugin.ConfigMutator = &Plugin{}

func (m *Plugin) Name() string {
	return "schemaconfig"
}

func (m *Plugin) MutateConfig(cfg *config.Config) error {
	if err := cfg.Check(); err != nil {
		return err
	}

	schema, _, err := cfg.LoadSchema()
	if err != nil {
		return err
	}

	cfg.Directives["goModel"] = config.DirectiveConfig{
		SkipRuntime: true,
	}

	cfg.Directives["goField"] = config.DirectiveConfig{
		SkipRuntime: true,
	}

	for _, schemaType := range schema.Types {
		if schemaType == schema.Query || schemaType == schema.Mutation || schemaType == schema.Subscription {
			continue
		}

		if bd := schemaType.Directives.ForName("goModel"); bd != nil {
			if ma := bd.Arguments.ForName("model"); ma != nil {
				if mv, err := ma.Value.Value(nil); err == nil {
					cfg.Models.Add(schemaType.Name, mv.(string))
				}
			}
			if ma := bd.Arguments.ForName("models"); ma != nil {
				if mvs, err := ma.Value.Value(nil); err == nil {
					for _, mv := range mvs.([]interface{}) {
						cfg.Models.Add(schemaType.Name, mv.(string))
					}
				}
			}
		}

		if schemaType.Kind == ast.Object || schemaType.Kind == ast.InputObject {
			for _, field := range schemaType.Fields {
				if fd := field.Directives.ForName("goField"); fd != nil {
					forceResolver := cfg.Models[schemaType.Name].Fields[field.Name].Resolver
					fieldName := cfg.Models[schemaType.Name].Fields[field.Name].FieldName

					if ra := fd.Arguments.ForName("forceResolver"); ra != nil {
						if fr, err := ra.Value.Value(nil); err == nil {
							forceResolver = fr.(bool)
						}
					}

					if na := fd.Arguments.ForName("name"); na != nil {
						if fr, err := na.Value.Value(nil); err == nil {
							fieldName = fr.(string)
						}
					}

					if cfg.Models[schemaType.Name].Fields == nil {
						cfg.Models[schemaType.Name] = config.TypeMapEntry{
							Model:  cfg.Models[schemaType.Name].Model,
							Fields: map[string]config.TypeMapField{},
						}
					}

					cfg.Models[schemaType.Name].Fields[field.Name] = config.TypeMapField{
						FieldName: fieldName,
						Resolver:  forceResolver,
					}
				}
			}
		}
	}
	return nil
}
