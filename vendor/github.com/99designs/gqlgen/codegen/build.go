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
	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	SchemaRaw        map[string]string
	SchemaFilename   SchemaFilenames
	Directives       []*Directive
}

type ModelBuild struct {
	PackageName string
	Models      []Model
	Enums       []Enum
}

type ResolverBuild struct {
	PackageName   string
	ResolverType  string
	Objects       Objects
	ResolverFound bool
}

type ServerBuild struct {
	PackageName         string
	ExecPackageName     string
	ResolverPackageName string
}

// Create a list of models that need to be generated
func (cfg *Config) models() (*ModelBuild, error) {
	namedTypes := cfg.buildNamedTypes()

	progLoader := cfg.newLoaderWithoutErrors()

	prog, err := progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	cfg.bindTypes(namedTypes, cfg.Model.Dir(), prog)

	models, err := cfg.buildModels(namedTypes, prog)
	if err != nil {
		return nil, err
	}
	return &ModelBuild{
		PackageName: cfg.Model.Package,
		Models:      models,
		Enums:       cfg.buildEnums(namedTypes),
	}, nil
}

// bind a schema together with some code to generate a Build
func (cfg *Config) resolver() (*ResolverBuild, error) {
	progLoader := cfg.newLoaderWithoutErrors()
	progLoader.Import(cfg.Resolver.ImportPath())

	prog, err := progLoader.Load()
	if err != nil {
		return nil, err
	}

	destDir := cfg.Resolver.Dir()

	namedTypes := cfg.buildNamedTypes()

	cfg.bindTypes(namedTypes, destDir, prog)

	objects, err := cfg.buildObjects(namedTypes, prog)
	if err != nil {
		return nil, err
	}

	def, _ := findGoType(prog, cfg.Resolver.ImportPath(), cfg.Resolver.Type)
	resolverFound := def != nil

	return &ResolverBuild{
		PackageName:   cfg.Resolver.Package,
		Objects:       objects,
		ResolverType:  cfg.Resolver.Type,
		ResolverFound: resolverFound,
	}, nil
}

func (cfg *Config) server(destDir string) *ServerBuild {
	return &ServerBuild{
		PackageName:         cfg.Resolver.Package,
		ExecPackageName:     cfg.Exec.ImportPath(),
		ResolverPackageName: cfg.Resolver.ImportPath(),
	}
}

// bind a schema together with some code to generate a Build
func (cfg *Config) bind() (*Build, error) {
	namedTypes := cfg.buildNamedTypes()

	progLoader := cfg.newLoaderWithoutErrors()
	prog, err := progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	cfg.bindTypes(namedTypes, cfg.Exec.Dir(), prog)

	objects, err := cfg.buildObjects(namedTypes, prog)
	if err != nil {
		return nil, err
	}

	inputs, err := cfg.buildInputs(namedTypes, prog)
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
	progLoader := cfg.newLoaderWithErrors()
	_, err := progLoader.Load()
	return err
}

func (cfg *Config) newLoaderWithErrors() loader.Config {
	conf := loader.Config{}

	for _, pkg := range cfg.Models.referencedPackages() {
		conf.Import(pkg)
	}
	return conf
}

func (cfg *Config) newLoaderWithoutErrors() loader.Config {
	conf := cfg.newLoaderWithErrors()
	conf.AllowErrors = true
	conf.TypeChecker = types.Config{
		Error: func(e error) {},
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
