package commands

import (
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/webui"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
	"log"
	"net/http"
)

func runWebUI(repo repository.Repo, args []string) error {
	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	router := mux.NewRouter()
	router.PathPrefix("/").Handler(http.FileServer(webui.WebUIAssets))

	open.Run(fmt.Sprintf("http://%s", addr))

	log.Fatal(http.ListenAndServe(addr, router))

	return nil
}

var webUICmd = &Command{
	Description: "Launch the web UI",
	Usage:       "",
	flagSet:     nil,
	RunMethod:   runWebUI,
}
