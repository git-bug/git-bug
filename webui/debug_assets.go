// +build !deploy_build

package webui

import "net/http"

var WebUIAssets http.FileSystem = http.Dir("webui/build")
