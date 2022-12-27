package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/webui"
)

const webUIOpenConfigKey = "git-bug.webui.open"

type webUIOptions struct {
	host      string
	port      int
	open      bool
	noOpen    bool
	readOnly  bool
	logErrors bool
	query     string
}

func newWebUICommand() *cobra.Command {
	env := execenv.NewEnv()
	options := webUIOptions{}

	cmd := &cobra.Command{
		Use:   "webui",
		Short: "Launch the web UI",
		Long: `Launch the web UI.

Available git config:
  git-bug.webui.open [bool]: control the automatic opening of the web UI in the default browser
`,
		PreRunE: execenv.LoadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebUI(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&options.host, "host", "127.0.0.1", "Network address or hostname to listen to (default to 127.0.0.1)")
	flags.BoolVar(&options.open, "open", false, "Automatically open the web UI in the default browser")
	flags.BoolVar(&options.noOpen, "no-open", false, "Prevent the automatic opening of the web UI in the default browser")
	flags.IntVarP(&options.port, "port", "p", 0, "Port to listen to (default to random available port)")
	flags.BoolVar(&options.readOnly, "read-only", false, "Whether to run the web UI in read-only mode")
	flags.BoolVar(&options.logErrors, "log-errors", false, "Whether to log errors")
	flags.StringVarP(&options.query, "query", "q", "", "The query to open in the web UI bug list")

	return cmd
}

func runWebUI(env *execenv.Env, opts webUIOptions) error {
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
		author, err := identity.GetUserIdentity(env.Repo)
		if err != nil {
			return err
		}
		router.Use(auth.Middleware(author.Id()))
	}

	mrc := cache.NewMultiRepoCache()
	_, events, err := mrc.RegisterDefaultRepository(env.Repo)
	if err != nil {
		return err
	}

	for event := range events {
		if event.Err != nil {
			env.Err.Printf("Cache building error [%s]: %v\n", event.Typename, event.Err)
			continue
		}
		switch event.Event {
		case cache.BuildEventCacheIsBuilt:
			env.Err.Println("Building cache... ")
		case cache.BuildEventStarted:
			env.Err.Printf("[%s] started\n", event.Typename)
		case cache.BuildEventFinished:
			env.Err.Printf("[%s] done\n", event.Typename)
		}
	}

	var errOut io.Writer
	if opts.logErrors {
		errOut = env.Err
	}

	graphqlHandler := graphql.NewHandler(mrc, errOut)

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
		env.Out.Println("WebUI is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the WebUI: %v\n", err)
		}

		// Teardown
		err := graphqlHandler.Close()
		if err != nil {
			env.Out.Println(err)
		}

		close(done)
	}()

	env.Out.Printf("Web UI: %s\n", webUiAddr)
	env.Out.Printf("Graphql API: http://%s/graphql\n", addr)
	env.Out.Printf("Graphql Playground: http://%s/playground\n", addr)
	env.Out.Println("Press Ctrl+c to quit")

	configOpen, err := env.Repo.AnyConfig().ReadBool(webUIOpenConfigKey)
	if errors.Is(err, repository.ErrNoConfigEntry) {
		// default to true
		configOpen = true
	} else if err != nil {
		return err
	}

	shouldOpen := (configOpen && !opts.noOpen) || opts.open

	if shouldOpen {
		err = open.Run(toOpen)
		if err != nil {
			env.Out.Println(err)
		}
	}

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-done

	env.Out.Println("WebUI stopped")
	return nil
}
