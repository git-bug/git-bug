package codegen

import (
	"fmt"
	"go/build"
	"sort"
	"strconv"
	"strings"
)

// These imports are referenced by the generated code, and are assumed to have the
// default alias. So lets make sure they get added first, and any later collisions get
// renamed.
var ambientImports = []string{
	"context",
	"fmt",
	"io",
	"strconv",
	"time",
	"sync",
	"github.com/vektah/gqlgen/neelance/introspection",
	"github.com/vektah/gqlgen/neelance/errors",
	"github.com/vektah/gqlgen/neelance/query",
	"github.com/vektah/gqlgen/neelance/schema",
	"github.com/vektah/gqlgen/neelance/validation",
	"github.com/vektah/gqlgen/graphql",
}

func buildImports(types NamedTypes, destDir string) *Imports {
	imports := Imports{
		destDir: destDir,
	}

	for _, ambient := range ambientImports {
		imports.add(ambient)
	}

	// Imports from top level user types
	for _, t := range types {
		t.Import = imports.add(t.Package)
	}

	return &imports
}

func (s *Imports) add(path string) *Import {
	if path == "" {
		return nil
	}

	if stringHasSuffixFold(s.destDir, path) {
		return nil
	}

	if existing := s.findByPath(path); existing != nil {
		return existing
	}

	pkg, err := build.Default.Import(path, s.destDir, 0)
	if err != nil {
		panic(err)
	}

	imp := &Import{
		Name: pkg.Name,
		Path: path,
	}
	s.imports = append(s.imports, imp)

	return imp
}

func stringHasSuffixFold(s, suffix string) bool {
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

func (s Imports) finalize() []*Import {
	// ensure stable ordering by sorting
	sort.Slice(s.imports, func(i, j int) bool {
		return s.imports[i].Path > s.imports[j].Path
	})

	for _, imp := range s.imports {
		alias := imp.Name

		i := 1
		for s.findByAlias(alias) != nil {
			alias = imp.Name + strconv.Itoa(i)
			i++
			if i > 10 {
				panic(fmt.Errorf("too many collisions, last attempt was %s", alias))
			}
		}
		imp.alias = alias
	}

	return s.imports
}

func (s Imports) findByPath(importPath string) *Import {
	for _, imp := range s.imports {
		if imp.Path == importPath {
			return imp
		}
	}
	return nil
}

func (s Imports) findByAlias(alias string) *Import {
	for _, imp := range s.imports {
		if imp.alias == alias {
			return imp
		}
	}
	return nil
}
