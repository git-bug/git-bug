package webui

import (
	"net/http"
	"os"
)

// implement a http.FileSystem that will serve a default file when the looked up
// file doesn't exist. Useful for Single-Page App that implement routing client
// side, where the server has to return the root index.html file for every route.
type fileSystemWithDefault struct {
	http.FileSystem
	defaultFile string
}

func (fswd *fileSystemWithDefault) Open(name string) (http.File, error) {
	f, err := fswd.FileSystem.Open(name)
	if os.IsNotExist(err) {
		return fswd.FileSystem.Open(fswd.defaultFile)
	}
	return f, err
}

func NewHandler() http.Handler {
	assetsHandler := &fileSystemWithDefault{
		FileSystem:  WebUIAssets,
		defaultFile: "index.html",
	}

	return http.FileServer(assetsHandler)
}
