// Wrapper around x/tools/imports that only removes imports, never adds new ones.

package imports

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"

	"golang.org/x/tools/go/ast/astutil"
)

type visitFn func(node ast.Node)

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	fn(node)
	return fn
}

// Prune removes any unused imports
func Prune(filename string, src []byte) ([]byte, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}

	unused, err := getUnusedImports(file, filename)
	if err != nil {
		return nil, err
	}
	for ipath, name := range unused {
		astutil.DeleteNamedImport(fset, file, name, ipath)
	}
	printConfig := &printer.Config{Mode: printer.TabIndent, Tabwidth: 8}

	var buf bytes.Buffer
	if err := printConfig.Fprint(&buf, fset, file); err != nil {
		return nil, err
	}

	return imports.Process(filename, buf.Bytes(), &imports.Options{FormatOnly: true, Comments: true, TabIndent: true, TabWidth: 8})
}

func getUnusedImports(file ast.Node, filename string) (map[string]string, error) {
	imported := map[string]*ast.ImportSpec{}
	used := map[string]bool{}

	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	srcDir := filepath.Dir(abs)

	ast.Walk(visitFn(func(node ast.Node) {
		if node == nil {
			return
		}
		switch v := node.(type) {
		case *ast.ImportSpec:
			if v.Name != nil {
				imported[v.Name.Name] = v
				break
			}
			ipath := strings.Trim(v.Path.Value, `"`)
			if ipath == "C" {
				break
			}

			local := importPathToName(ipath, srcDir)

			imported[local] = v
		case *ast.SelectorExpr:
			xident, ok := v.X.(*ast.Ident)
			if !ok {
				break
			}
			if xident.Obj != nil {
				// if the parser can resolve it, it's not a package ref
				break
			}
			used[xident.Name] = true
		}
	}), file)

	for pkg := range used {
		delete(imported, pkg)
	}

	unusedImport := map[string]string{}
	for pkg, is := range imported {
		if !used[pkg] && pkg != "_" && pkg != "." {
			name := ""
			if is.Name != nil {
				name = is.Name.Name
			}
			unusedImport[strings.Trim(is.Path.Value, `"`)] = name
		}
	}

	return unusedImport, nil
}

func importPathToName(importPath, srcDir string) (packageName string) {
	pkg, err := build.Default.Import(importPath, srcDir, 0)
	if err != nil {
		return ""
	}

	return pkg.Name
}
