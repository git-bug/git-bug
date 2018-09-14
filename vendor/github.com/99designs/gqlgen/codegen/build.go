package codegen

import (
	"fmt"
	"go/build"
	"go/types"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

type Build struct {
	PackageName      string
	Objects          Objects
	Inputs           Objects
	Interfaces       []*Interface
	Imports          []*Import
	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	SchemaRaw        string
	SchemaFilename   string
	Directives       []*Directive
}

type ModelBuild struct {
	PackageName string
	Imports     []*Import
	Models      []Model
	Enums       []Enum
}

type ResolverBuild struct {
	PackageName   string
	Imports       []*Import
	ResolverType  string
	Objects       Objects
	ResolverFound bool
}

type ServerBuild struct {
	PackageName         string
	Imports             []*Import
	ExecPackageName     string
	ResolverPackageName string
}

// Create a list of models that need to be generated
func (cfg *Config) models() (*ModelBuild, error) {
	namedTypes := cfg.buildNamedTypes()

	progLoader := newLoader(namedTypes, true)
	prog, err := progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}
	imports := buildImports(namedTypes, cfg.Model.Dir())

	cfg.bindTypes(imports, namedTypes, cfg.Model.Dir(), prog)

	models, err := cfg.buildModels(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}
	return &ModelBuild{
		PackageName: cfg.Model.Package,
		Models:      models,
		Enums:       cfg.buildEnums(namedTypes),
		Imports:     imports.finalize(),
	}, nil
}

// bind a schema together with some code to generate a Build
func (cfg *Config) resolver() (*ResolverBuild, error) {
	progLoader := newLoader(cfg.buildNamedTypes(), true)
	progLoader.Import(cfg.Resolver.ImportPath())

	prog, err := progLoader.Load()
	if err != nil {
		return nil, err
	}

	destDir := cfg.Resolver.Dir()

	namedTypes := cfg.buildNamedTypes()
	imports := buildImports(namedTypes, destDir)
	imports.add(cfg.Exec.ImportPath())
	imports.add("github.com/99designs/gqlgen/handler") // avoid import github.com/vektah/gqlgen/handler

	cfg.bindTypes(imports, namedTypes, destDir, prog)

	objects, err := cfg.buildObjects(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}

	def, _ := findGoType(prog, cfg.Resolver.ImportPath(), cfg.Resolver.Type)
	resolverFound := def != nil

	return &ResolverBuild{
		PackageName:   cfg.Resolver.Package,
		Imports:       imports.finalize(),
		Objects:       objects,
		ResolverType:  cfg.Resolver.Type,
		ResolverFound: resolverFound,
	}, nil
}

func (cfg *Config) server(destDir string) *ServerBuild {
	imports := buildImports(NamedTypes{}, destDir)
	imports.add(cfg.Exec.ImportPath())
	imports.add(cfg.Resolver.ImportPath())

	return &ServerBuild{
		PackageName:         cfg.Resolver.Package,
		Imports:             imports.finalize(),
		ExecPackageName:     cfg.Exec.Package,
		ResolverPackageName: cfg.Resolver.Package,
	}
}

// bind a schema together with some code to generate a Build
func (cfg *Config) bind() (*Build, error) {
	namedTypes := cfg.buildNamedTypes()

	progLoader := newLoader(namedTypes, true)
	prog, err := progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	imports := buildImports(namedTypes, cfg.Exec.Dir())
	cfg.bindTypes(imports, namedTypes, cfg.Exec.Dir(), prog)

	objects, err := cfg.buildObjects(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}

	inputs, err := cfg.buildInputs(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}
	directives, err := cfg.buildDirectives(namedTypes)
	if err != nil {
		return nil, err
	}

	b := &Build{
		PackageName:    cfg.Exec.Package,
		Objects:        objects,
		Interfaces:     cfg.buildInterfaces(namedTypes, prog),
		Inputs:         inputs,
		Imports:        imports.finalize(),
		SchemaRaw:      cfg.SchemaStr,
		SchemaFilename: cfg.SchemaFilename,
		Directives:     directives,
	}

	if cfg.schema.Query != nil {
		b.QueryRoot = b.Objects.ByName(cfg.schema.Query.Name)
	} else {
		return b, fmt.Errorf("query entry point missing")
	}

	if cfg.schema.Mutation != nil {
		b.MutationRoot = b.Objects.ByName(cfg.schema.Mutation.Name)
	}

	if cfg.schema.Subscription != nil {
		b.SubscriptionRoot = b.Objects.ByName(cfg.schema.Subscription.Name)
	}
	return b, nil
}

func (cfg *Config) validate() error {
	progLoader := newLoader(cfg.buildNamedTypes(), false)
	_, err := progLoader.Load()
	return err
}

func newLoader(namedTypes NamedTypes, allowErrors bool) loader.Config {
	conf := loader.Config{}
	if allowErrors {
		conf = loader.Config{
			AllowErrors: true,
			TypeChecker: types.Config{
				Error: func(e error) {},
			},
		}
	}
	for _, imp := range ambientImports {
		conf.Import(imp)
	}

	for _, imp := range namedTypes {
		if imp.Package != "" {
			conf.Import(imp.Package)
		}
	}
	return conf
}

func resolvePkg(pkgName string) (string, error) {
	cwd, _ := os.Getwd()

	pkg, err := build.Default.Import(pkgName, cwd, build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.ImportPath, nil
}
