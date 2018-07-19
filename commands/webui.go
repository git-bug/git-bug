package commands

import (
	"fmt"
	"github.com/MichaelMure/git-bug/graphql"
	"github.com/MichaelMure/git-bug/webui"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

func runWebUI(cmd *cobra.Command, args []string) error {
	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	webUiAddr := fmt.Sprintf("http://%s", addr)

	fmt.Printf("Web UI available at %s\n", webUiAddr)

	graphqlHandler, err := graphql.NewHandler()

	if err != nil {
		return err
	}

	router := mux.NewRouter()

	// Routes
	router.Path("/graphql").Handler(graphqlHandler)
	router.PathPrefix("/").Handler(http.FileServer(webui.WebUIAssets))

	open.Run(webUiAddr)

	log.Fatal(http.ListenAndServe(addr, router))

	return nil
}

var webUICmd = &cobra.Command{
	Use:   "webui",
	Short: "Launch the web UI",
	RunE:  runWebUI,
}

func init() {
	rootCmd.AddCommand(webUICmd)
}
