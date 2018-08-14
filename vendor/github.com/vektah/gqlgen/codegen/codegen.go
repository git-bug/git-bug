package codegen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/codegen/templates"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/imports"
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
		var buf *bytes.Buffer
		buf, err = templates.Run("models.gotpl", modelsBuild)
		if err != nil {
			return errors.Wrap(err, "model generation failed")
		}

		if err = write(cfg.Model.Filename, buf.Bytes()); err != nil {
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

	var buf *bytes.Buffer
	buf, err = templates.Run("generated.gotpl", build)
	if err != nil {
		return errors.Wrap(err, "exec codegen failed")
	}

	if err = write(cfg.Exec.Filename, buf.Bytes()); err != nil {
		return err
	}

	if err = cfg.validate(); err != nil {
		return errors.Wrap(err, "validation failed")
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

	builtins := TypeMap{
		"__Directive":  {Model: "github.com/vektah/gqlgen/neelance/introspection.Directive"},
		"__Type":       {Model: "github.com/vektah/gqlgen/neelance/introspection.Type"},
		"__Field":      {Model: "github.com/vektah/gqlgen/neelance/introspection.Field"},
		"__EnumValue":  {Model: "github.com/vektah/gqlgen/neelance/introspection.EnumValue"},
		"__InputValue": {Model: "github.com/vektah/gqlgen/neelance/introspection.InputValue"},
		"__Schema":     {Model: "github.com/vektah/gqlgen/neelance/introspection.Schema"},
		"Int":          {Model: "github.com/vektah/gqlgen/graphql.Int"},
		"Float":        {Model: "github.com/vektah/gqlgen/graphql.Float"},
		"String":       {Model: "github.com/vektah/gqlgen/graphql.String"},
		"Boolean":      {Model: "github.com/vektah/gqlgen/graphql.Boolean"},
		"ID":           {Model: "github.com/vektah/gqlgen/graphql.ID"},
		"Time":         {Model: "github.com/vektah/gqlgen/graphql.Time"},
		"Map":          {Model: "github.com/vektah/gqlgen/graphql.Map"},
	}

	if cfg.Models == nil {
		cfg.Models = TypeMap{}
	}
	for typeName, entry := range builtins {
		if !cfg.Models.Exists(typeName) {
			cfg.Models[typeName] = entry
		}
	}

	cfg.schema = schema.New()
	return cfg.schema.Parse(cfg.SchemaStr)
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

func gofmt(filename string, b []byte) ([]byte, error) {
	out, err := imports.Process(filename, b, nil)
	if err != nil {
		return b, errors.Wrap(err, "unable to gofmt")
	}
	return out, nil
}

func write(filename string, b []byte) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	formatted, err := gofmt(filename, b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt failed: %s\n", err.Error())
		formatted = b
	}

	err = ioutil.WriteFile(filename, formatted, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", filename)
	}

	return nil
}
