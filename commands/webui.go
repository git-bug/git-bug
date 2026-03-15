package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/api/graphql"
	httpapi "github.com/git-bug/git-bug/api/http"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/webui"
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

func newWebUICommand(env *execenv.Env) *cobra.Command {
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

	_, events := mrc.RegisterDefaultRepository(env.Repo)

	err := execenv.CacheBuildProgressBar(env, events)
	if err != nil {
		return err
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

	go func() {
		<-env.Ctx.Done()
		env.Out.Println("shutting down...")

		ctxTeardown, cancelTeardown := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelTeardown()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctxTeardown); err != nil {
			env.Err.Printf("Could not gracefully shutdown the WebUI: %v\n", err)
		}

		// Teardown
		err := graphqlHandler.Close()
		if err != nil {
			env.Err.Println(err)
		}

		close(done)
	}()

	env.Out.Printf("Web UI: %s\n", webUiAddr)
	env.Out.Printf("Graphql API: http://%s/graphql\n", addr)
	env.Out.Printf("Graphql Playground: http://%s/playground\n", addr)
	env.Out.Printf("\n[ Press Ctrl+c to quit ]\n\n")

	configOpen, err := env.Repo.AnyConfig().ReadBool(webUIOpenConfigKey)
	if errors.Is(err, repository.ErrNoConfigEntry) {
		// default to true
		configOpen = true
	} else if err != nil {
		return err
	}

	shouldOpen := (configOpen && !opts.noOpen) || opts.open

	if shouldOpen {
		go func() {
			const maxAttempts = 3
			if isUp(toOpen, maxAttempts, 3*time.Second) {
				err = open.Run(toOpen)
				if err != nil {
					env.Err.Println(err)
					return
				}

				env.Out.Printf("opened your default browser to url: %s\n", toOpen)
				return
			}

			env.Err.Printf(
				"uh oh! it appears that the http server hasn't started.\n"+
					"we failed to reach %s after %d attempts, exiting now.\n",
				toOpen, maxAttempts,
			)
		}()
	}

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-done
	return nil
}

func isUp(url string, maxRetries int, initialDelay time.Duration) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	delay := initialDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := client.Head(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return true
			}
		}

		if attempt < maxRetries {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return false
}
