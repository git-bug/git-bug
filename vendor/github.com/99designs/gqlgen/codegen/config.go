package codegen

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/internal/gopath"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"gopkg.in/yaml.v2"
)

var cfgFilenames = []string{".gqlgen.yml", "gqlgen.yml", "gqlgen.yaml"}

// DefaultConfig creates a copy of the default config
func DefaultConfig() *Config {
	return &Config{
		SchemaFilename: SchemaFilenames{"schema.graphql"},
		SchemaStr:      map[string]string{},
		Model:          PackageConfig{Filename: "models_gen.go"},
		Exec:           PackageConfig{Filename: "generated.go"},
	}
}

// LoadConfigFromDefaultLocations looks for a config file in the current directory, and all parent directories
// walking up the tree. The closest config file will be returned.
func LoadConfigFromDefaultLocations() (*Config, error) {
	cfgFile, err := findCfg()
	if err != nil {
		return nil, err
	}

	err = os.Chdir(filepath.Dir(cfgFile))
	if err != nil {
		return nil, errors.Wrap(err, "unable to enter config dir")
	}
	return LoadConfig(cfgFile)
}

// LoadConfig reads the gqlgen.yml config file
func LoadConfig(filename string) (*Config, error) {
	config := DefaultConfig()

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read config")
	}

	if err := yaml.UnmarshalStrict(b, config); err != nil {
		return nil, errors.Wrap(err, "unable to parse config")
	}

	preGlobbing := config.SchemaFilename
	config.SchemaFilename = SchemaFilenames{}
	for _, f := range preGlobbing {
		matches, err := filepath.Glob(f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to glob schema filename %s", f)
		}

		for _, m := range matches {
			if config.SchemaFilename.Has(m) {
				continue
			}
			config.SchemaFilename = append(config.SchemaFilename, m)
		}
	}

	config.FilePath = filename
	config.SchemaStr = map[string]string{}

	return config, nil
}

type Config struct {
	SchemaFilename SchemaFilenames   `yaml:"schema,omitempty"`
	SchemaStr      map[string]string `yaml:"-"`
	Exec           PackageConfig     `yaml:"exec"`
	Model          PackageConfig     `yaml:"model"`
	Resolver       PackageConfig     `yaml:"resolver,omitempty"`
	Models         TypeMap           `yaml:"models,omitempty"`
	StructTag      string            `yaml:"struct_tag,omitempty"`

	FilePath string `yaml:"-"`

	schema *ast.Schema `yaml:"-"`
}

type PackageConfig struct {
	Filename string `yaml:"filename,omitempty"`
	Package  string `yaml:"package,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

type TypeMapEntry struct {
	Model  string                  `yaml:"model"`
	Fields map[string]TypeMapField `yaml:"fields,omitempty"`
}

type TypeMapField struct {
	Resolver  bool   `yaml:"resolver"`
	FieldName string `yaml:"fieldName"`
}

type SchemaFilenames []string

func (a *SchemaFilenames) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	err := unmarshal(&single)
	if err == nil {
		*a = []string{single}
		return nil
	}

	var multi []string
	err = unmarshal(&multi)
	if err != nil {
		return err
	}

	*a = multi
	return nil
}

func (a SchemaFilenames) Has(file string) bool {
	for _, existing := range a {
		if existing == file {
			return true
		}
	}
	return false
}

func (c *PackageConfig) normalize() error {
	if c.Filename == "" {
		return errors.New("Filename is required")
	}
	c.Filename = abs(c.Filename)
	// If Package is not set, first attempt to load the package at the output dir. If that fails
	// fallback to just the base dir name of the output filename.
	if c.Package == "" {
		cwd, _ := os.Getwd()
		pkg, _ := build.Default.Import(c.ImportPath(), cwd, 0)
		if pkg.Name != "" {
			c.Package = pkg.Name
		} else {
			c.Package = filepath.Base(c.Dir())
		}
	}
	c.Package = sanitizePackageName(c.Package)
	return nil
}

func (c *PackageConfig) ImportPath() string {
	return gopath.MustDir2Import(c.Dir())
}

func (c *PackageConfig) Dir() string {
	return filepath.Dir(c.Filename)
}

func (c *PackageConfig) Check() error {
	if strings.ContainsAny(c.Package, "./\\") {
		return fmt.Errorf("package should be the output package name only, do not include the output filename")
	}
	if c.Filename != "" && !strings.HasSuffix(c.Filename, ".go") {
		return fmt.Errorf("filename should be path to a go source file")
	}
	return nil
}

func (c *PackageConfig) IsDefined() bool {
	return c.Filename != ""
}

func (cfg *Config) Check() error {
	if err := cfg.Models.Check(); err != nil {
		return errors.Wrap(err, "config.models")
	}
	if err := cfg.Exec.Check(); err != nil {
		return errors.Wrap(err, "config.exec")
	}
	if err := cfg.Model.Check(); err != nil {
		return errors.Wrap(err, "config.model")
	}
	if err := cfg.Resolver.Check(); err != nil {
		return errors.Wrap(err, "config.resolver")
	}
	return nil
}

type TypeMap map[string]TypeMapEntry

func (tm TypeMap) Exists(typeName string) bool {
	_, ok := tm[typeName]
	return ok
}

func (tm TypeMap) Check() error {
	for typeName, entry := range tm {
		if strings.LastIndex(entry.Model, ".") < strings.LastIndex(entry.Model, "/") {
			return fmt.Errorf("model %s: invalid type specifier \"%s\" - you need to specify a struct to map to", typeName, entry.Model)
		}
	}
	return nil
}

func (tm TypeMap) referencedPackages() []string {
	var pkgs []string

	for _, typ := range tm {
		if typ.Model == "map[string]interface{}" {
			continue
		}
		pkg, _ := pkgAndType(typ.Model)
		if pkg == "" || inStrSlice(pkgs, pkg) {
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i] > pkgs[j]
	})
	return pkgs
}

func inStrSlice(haystack []string, needle string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}

	return false
}

// findCfg searches for the config file in this directory and all parents up the tree
// looking for the closest match
func findCfg() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get working dir to findCfg")
	}

	cfg := findCfgInDir(dir)

	for cfg == "" && dir != filepath.Dir(dir) {
		dir = filepath.Dir(dir)
		cfg = findCfgInDir(dir)
	}

	if cfg == "" {
		return "", os.ErrNotExist
	}

	return cfg, nil
}

func findCfgInDir(dir string) string {
	for _, cfgName := range cfgFilenames {
		path := filepath.Join(dir, cfgName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
