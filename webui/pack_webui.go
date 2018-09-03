// +build ignore

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/vfsgen"
)

func main() {
	var cwd, _ = os.Getwd()

	webUIAssets := filter.Skip(
		http.Dir(filepath.Join(cwd, "webui/build")),
		func(path string, fi os.FileInfo) bool {
			return filter.FilesWithExtensions(".map")(path, fi)
		},
	)

	fmt.Println("Packing Web UI files ...")

	err := vfsgen.Generate(webUIAssets, vfsgen.Options{
		Filename:     "webui/packed_assets.go",
		PackageName:  "webui",
		BuildTags:    "!debugwebui",
		VariableName: "WebUIAssets",
	})

	if err != nil {
		log.Fatalln(err)
	}
}
