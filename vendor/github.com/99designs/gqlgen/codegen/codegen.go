package codegen

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
)

func Generate(cfg Config) error {
	if err := cfg.normalize(); err != nil {
		return err
	}

	_ = syscall.Unlink(cfg.Exec.Filename)
	_ = syscall.Unlink(cfg.Model.Filename)

	modelsBuild, err := cfg.models()
	if err != nil {
		return errors.Wrap(err, "model plan failed")
	}
	if len(modelsBuild.Models) > 0 || len(modelsBuild.Enums) > 0 {
		if err = templates.RenderToFile("models.gotpl", cfg.Model.Filename, modelsBuild); err != nil {
			return err
		}

		for _, model := range modelsBuild.Models {
			modelCfg := cfg.Models[model.GQLType]
			modelCfg.Model = cfg.Model.ImportPath() + "." + model.GoType
			cfg.Models[model.GQLType] = modelCfg
		}

		for _, enum := range modelsBuild.Enums {
			modelCfg := cfg.Models[enum.GQLType]
			modelCfg.Model = cfg.Model.ImportPath() + "." + enum.GoType
			cfg.Models[enum.GQLType] = modelCfg
		}
	}

	build, err := cfg.bind()
	if err != nil {
		return errors.Wrap(err, "exec plan failed")
	}

	if err := templates.RenderToFile("generated.gotpl", cfg.Exec.Filename, build); err != nil {
		return err
	}

	if cfg.Resolver.IsDefined() {
		if err := generateResolver(cfg); err != nil {
			return errors.Wrap(err, "generating resolver failed")
		}
	}

	if err := cfg.validate(); err != nil {
		return errors.Wrap(err, "validation failed")
	}

	return nil
}

func GenerateServer(cfg Config, filename string) error {
	if err := cfg.Exec.normalize(); err != nil {
		return errors.Wrap(err, "exec")
	}
	if err := cfg.Resolver.normalize(); err != nil {
		return errors.Wrap(err, "resolver")
	}

	serverFilename := abs(filename)
	serverBuild := cfg.server(filepath.Dir(serverFilename))

	if _, err := os.Stat(serverFilename); os.IsNotExist(errors.Cause(err)) {
		err = templates.RenderToFile("server.gotpl", serverFilename, serverBuild)
		if err != nil {
			return errors.Wrap(err, "generate server failed")
		}
	} else {
		log.Printf("Skipped server: %s already exists\n", serverFilename)
	}
	return nil
}

func generateResolver(cfg Config) error {
	resolverBuild, err := cfg.resolver()
	if err != nil {
		return errors.Wrap(err, "resolver build failed")
	}
	filename := cfg.Resolver.Filename

	if resolverBuild.ResolverFound {
		log.Printf("Skipped resolver: %s.%s already exists\n", cfg.Resolver.ImportPath(), cfg.Resolver.Type)
		return nil
	}

	if _, err := os.Stat(filename); os.IsNotExist(errors.Cause(err)) {
		if err := templates.RenderToFile("resolver.gotpl", filename, resolverBuild); err != nil {
			return err
		}
	} else {
		log.Printf("Skipped resolver: %s already exists\n", filename)
	}

	return nil
}

func (cfg *Config) normalize() error {
	if err := cfg.Model.normalize(); err != nil {
		return errors.Wrap(err, "model")
	}

	if err := cfg.Exec.normalize(); err != nil {
		return errors.Wrap(err, "exec")
	}

	if cfg.Resolver.IsDefined() {
		if err := cfg.Resolver.normalize(); err != nil {
			return errors.Wrap(err, "resolver")
		}
	}

	builtins := TypeMap{
		"__Directive":  {Model: "github.com/99designs/gqlgen/graphql/introspection.Directive"},
		"__Type":       {Model: "github.com/99designs/gqlgen/graphql/introspection.Type"},
		"__Field":      {Model: "github.com/99designs/gqlgen/graphql/introspection.Field"},
		"__EnumValue":  {Model: "github.com/99designs/gqlgen/graphql/introspection.EnumValue"},
		"__InputValue": {Model: "github.com/99designs/gqlgen/graphql/introspection.InputValue"},
		"__Schema":     {Model: "github.com/99designs/gqlgen/graphql/introspection.Schema"},
		"Int":          {Model: "github.com/99designs/gqlgen/graphql.Int"},
		"Float":        {Model: "github.com/99designs/gqlgen/graphql.Float"},
		"String":       {Model: "github.com/99designs/gqlgen/graphql.String"},
		"Boolean":      {Model: "github.com/99designs/gqlgen/graphql.Boolean"},
		"ID":           {Model: "github.com/99designs/gqlgen/graphql.ID"},
		"Time":         {Model: "github.com/99designs/gqlgen/graphql.Time"},
		"Map":          {Model: "github.com/99designs/gqlgen/graphql.Map"},
	}

	if cfg.Models == nil {
		cfg.Models = TypeMap{}
	}
	for typeName, entry := range builtins {
		if !cfg.Models.Exists(typeName) {
			cfg.Models[typeName] = entry
		}
	}

	var sources []*ast.Source
	for _, filename := range cfg.SchemaFilename {
		sources = append(sources, &ast.Source{Name: filename, Input: cfg.SchemaStr[filename]})
	}

	var err *gqlerror.Error
	cfg.schema, err = gqlparser.LoadSchema(sources...)
	if err != nil {
		return err
	}
	return nil
}

var invalidPackageNameChar = regexp.MustCompile(`[^\w]`)

func sanitizePackageName(pkg string) string {
	return invalidPackageNameChar.ReplaceAllLiteralString(filepath.Base(pkg), "_")
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return filepath.ToSlash(absPath)
}
