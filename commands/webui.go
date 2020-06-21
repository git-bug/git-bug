package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/api/auth"
	"github.com/MichaelMure/git-bug/api/graphql"
	httpapi "github.com/MichaelMure/git-bug/api/http"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/webui"
)

var (
	webUIPort     int
	webUIOpen     bool
	webUINoOpen   bool
	webUIReadOnly bool
)

const webUIOpenConfigKey = "git-bug.webui.open"

func runWebUI(cmd *cobra.Command, args []string) error {
	if webUIPort == 0 {
		var err error
		webUIPort, err = freeport.GetFreePort()
		if err != nil {
			return err
		}
	}

	addr := fmt.Sprintf("127.0.0.1:%d", webUIPort)
	webUiAddr := fmt.Sprintf("http://%s", addr)

	router := mux.NewRouter()

	// If the webUI is not read-only, use an authentication middleware with a
	// fixed identity: the default user of the repo
	// TODO: support dynamic authentication with OAuth
	if !webUIReadOnly {
		author, err := identity.GetUserIdentity(repo)
		if err != nil {
			return err
		}
		router.Use(auth.Middleware(author.Id()))
	}

	mrc := cache.NewMultiRepoCache()
	_, err := mrc.RegisterDefaultRepository(repo)
	if err != nil {
		return err
	}

	graphqlHandler := graphql.NewHandler(mrc)

	// Routes
	router.Path("/playground").Handler(playground.Handler("git-bug", "/graphql"))
	router.Path("/graphql").Handler(graphqlHandler)
	router.Path("/gitfile/{repo}/{hash}").Handler(httpapi.NewGitFileHandler(mrc))
	router.Path("/upload/{repo}").Methods("POST").Handler(httpapi.NewGitUploadFileHandler(mrc))
	router.PathPrefix("/").Handler(webui.NewHandler())

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)

	// register as handler of the interrupt signal to trigger the teardown
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		fmt.Println("WebUI is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the WebUI: %v\n", err)
		}

		// Teardown
		err := graphqlHandler.Close()
		if err != nil {
			fmt.Println(err)
		}

		close(done)
	}()

	fmt.Printf("Web UI: %s\n", webUiAddr)
	fmt.Printf("Graphql API: http://%s/graphql\n", addr)
	fmt.Printf("Graphql Playground: http://%s/playground\n", addr)
	fmt.Println("Press Ctrl+c to quit")

	configOpen, err := repo.LocalConfig().ReadBool(webUIOpenConfigKey)
	if err == repository.ErrNoConfigEntry {
		// default to true
		configOpen = true
	} else if err != nil {
		return err
	}

	shouldOpen := (configOpen && !webUINoOpen) || webUIOpen

	if shouldOpen {
		err = open.Run(webUiAddr)
		if err != nil {
			fmt.Println(err)
		}
	}

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-done

	fmt.Println("WebUI stopped")
	return nil
}

var webUICmd = &cobra.Command{
	Use:   "webui",
	Short: "Launch the web UI.",
	Long: `Launch the web UI.

Available git config:
  git-bug.webui.open [bool]: control the automatic opening of the web UI in the default browser
`,
	PreRunE: loadRepo,
	RunE:    runWebUI,
}

func init() {
	RootCmd.AddCommand(webUICmd)

	webUICmd.Flags().SortFlags = false

	webUICmd.Flags().BoolVar(&webUIOpen, "open", false, "Automatically open the web UI in the default browser")
	webUICmd.Flags().BoolVar(&webUINoOpen, "no-open", false, "Prevent the automatic opening of the web UI in the default browser")
	webUICmd.Flags().IntVarP(&webUIPort, "port", "p", 0, "Port to listen to (default is random)")
	webUICmd.Flags().BoolVar(&webUIReadOnly, "read-only", false, "Whether to run the web UI in read-only mode")
}
