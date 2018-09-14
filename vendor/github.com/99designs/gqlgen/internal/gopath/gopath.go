package gopath

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"
)

var NotFound = fmt.Errorf("not on GOPATH")

// Contains returns true if the given directory is in the GOPATH
func Contains(dir string) bool {
	_, err := Dir2Import(dir)
	return err == nil
}

// Dir2Import takes an *absolute* path and returns a golang import path for the package, and returns an error if it isn't on the gopath
func Dir2Import(dir string) (string, error) {
	dir = filepath.ToSlash(dir)
	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		gopath = filepath.ToSlash(filepath.Join(gopath, "src"))
		if len(gopath) < len(dir) && strings.EqualFold(gopath, dir[0:len(gopath)]) {
			return dir[len(gopath)+1:], nil
		}
	}
	return "", NotFound
}

// MustDir2Import takes an *absolute* path and returns a golang import path for the package, and panics if it isn't on the gopath
func MustDir2Import(dir string) string {
	pkg, err := Dir2Import(dir)
	if err != nil {
		panic(err)
	}
	return pkg
}
