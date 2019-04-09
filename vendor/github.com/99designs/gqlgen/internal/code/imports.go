package code

import (
	"errors"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/packages"
)

var pathForDirCache = sync.Map{}

// ImportPathFromDir takes an *absolute* path and returns a golang import path for the package, and returns an error if it isn't on the gopath
func ImportPathForDir(dir string) string {
	if v, ok := pathForDirCache.Load(dir); ok {
		return v.(string)
	}

	p, _ := packages.Load(&packages.Config{
		Dir: dir,
	}, ".")

	// If the dir dosent exist yet, keep walking up the directory tree trying to find a match
	if len(p) != 1 {
		parent, err := filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			panic(err)
		}
		// Walked all the way to the root and didnt find anything :'(
		if parent == dir {
			return ""
		}
		return ImportPathForDir(parent) + "/" + filepath.Base(dir)
	}

	pathForDirCache.Store(dir, p[0].PkgPath)

	return p[0].PkgPath
}

var nameForPackageCache = sync.Map{}

func NameForPackage(importPath string) string {
	if importPath == "" {
		panic(errors.New("import path can not be empty"))
	}
	if v, ok := nameForPackageCache.Load(importPath); ok {
		return v.(string)
	}
	importPath = QualifyPackagePath(importPath)
	p, _ := packages.Load(nil, importPath)

	if len(p) != 1 || p[0].Name == "" {
		return SanitizePackageName(filepath.Base(importPath))
	}

	nameForPackageCache.Store(importPath, p[0].Name)

	return p[0].Name
}
