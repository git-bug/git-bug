package commands

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
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

const webUIOpenConfigKey = "git-bug.webui.open"

type webUIOptions struct {
	host     string
	port     int
	open     bool
	noOpen   bool
	readOnly bool
	query    string
}

func newWebUICommand() *cobra.Command {
	env := newEnv()
	options := webUIOptions{}

	cmd := &cobra.Command{
		Use:   "webui",
		Short: "Launch the web UI.",
		Long: `Launch the web UI.

Available git config:
  git-bug.webui.open [bool]: control the automatic opening of the web UI in the default browser
`,
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebUI(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&options.host, "host", "127.0.0.1", "Network address or hostname to listen to (default to 127.0.0.1)")
	flags.BoolVar(&options.open, "open", false, "Automatically open the web UI in the default browser")
	flags.BoolVar(&options.noOpen, "no-open", false, "Prevent the automatic opening of the web UI in the default browser")
	flags.IntVarP(&options.port, "port", "p", 0, "Port to listen to (default to random available port)")
	flags.BoolVar(&options.readOnly, "read-only", false, "Whether to run the web UI in read-only mode")
	flags.StringVarP(&options.query, "query", "q", "", "The query to open in the web UI bug list")

	return cmd
}

func runWebUI(env *Env, opts webUIOptions, args []string) error {
	if opts.port == 0 {
		var err error
		opts.port, err = freeport.GetFreePort()
		if err != nil {
			return err
		}
	}

	addr := net.JoinHostPort(opts.host, strconv.Itoa(opts.port))
	webUiAddr := fmt.Sprintf("http://%s", addr)
	toOpen := webUiAddr

	if len(opts.query) > 0 {
		// Explicitly set the query parameter instead of going with a default one.
		toOpen = fmt.Sprintf("%s/?q=%s", webUiAddr, url.QueryEscape(opts.query))
	}

	router := mux.NewRouter()

	// If the webUI is not read-only, use an authentication middleware with a
	// fixed identity: the default user of the repo
	// TODO: support dynamic authentication with OAuth
	if !opts.readOnly {
		author, err := identity.GetUserIdentity(env.repo)
		if err != nil {
			return err
		}
		router.Use(auth.Middleware(author.Id()))
	}

	mrc := cache.NewMultiRepoCache()
	_, err := mrc.RegisterDefaultRepository(env.repo)
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
		env.out.Println("WebUI is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the WebUI: %v\n", err)
		}

		// Teardown
		err := graphqlHandler.Close()
		if err != nil {
			env.out.Println(err)
		}

		close(done)
	}()

	env.out.Printf("Web UI: %s\n", webUiAddr)
	env.out.Printf("Graphql API: http://%s/graphql\n", addr)
	env.out.Printf("Graphql Playground: http://%s/playground\n", addr)
	env.out.Println("Press Ctrl+c to quit")

	configOpen, err := env.repo.AnyConfig().ReadBool(webUIOpenConfigKey)
	if err == repository.ErrNoConfigEntry {
		// default to true
		configOpen = true
	} else if err != nil {
		return err
	}

	shouldOpen := (configOpen && !opts.noOpen) || opts.open

	if shouldOpen {
		err = open.Run(toOpen)
		if err != nil {
			env.out.Println(err)
		}
	}

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-done

	env.out.Println("WebUI stopped")
	return nil
}
