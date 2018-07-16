// +build ignore

package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/vfsgen"
)

func main() {
	var cwd, _ = os.Getwd()

	webUIAssets := http.Dir(filepath.Join(cwd, "webui/build"))

	err := vfsgen.Generate(webUIAssets, vfsgen.Options{
		Filename:     "webui/packed_assets.go",
		PackageName:  "webui",
		BuildTags:    "deploy_build",
		VariableName: "WebUIAssets",
	})

	if err != nil {
		log.Fatalln(err)
	}
}
