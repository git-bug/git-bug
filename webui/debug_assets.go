// +build debugwebui

package webui

import "net/http"

// WebUIAssets give access to the files of the Web UI for a http handler
// This access is only used in a debug build to be able to edit the WebUI
// files without having to package them.
var WebUIAssets http.FileSystem = http.Dir("webui/build")
