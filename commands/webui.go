package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/api/graphql"
	httpapi "github.com/git-bug/git-bug/api/http"
	"github.com/git-bug/git-bug/api/repoctx"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/webui"
)

const webUIOpenConfigKey = "git-bug.webui.open"

type webUIOptions struct {
	bind      string
	port      int
	open      bool
	noOpen    bool
	readOnly  bool
	logErrors bool
	query     string
	// Multi-repo mode: register each path given to --repo (repeatable), or
	// scan --root for repos matching <root>/*/*/ (owner/name). When more than
	// one repo is registered, read-only is forced (auth currently needs one
	// user identity, which is per-repo).
	repos []string
	root  string
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

	flags.StringVar(&options.bind, "bind", "127.0.0.1", "Network address to bind to (default to 127.0.0.1)")
	flags.IntVarP(&options.port, "port", "p", 0, "Port to listen on (default to random available port)")
	flags.BoolVar(&options.open, "open", false, "Automatically open the web UI in the default browser")
	flags.BoolVar(&options.noOpen, "no-open", false, "Prevent the automatic opening of the web UI in the default browser")
	flags.BoolVar(&options.readOnly, "read-only", false, "Whether to run the web UI in read-only mode")
	flags.BoolVar(&options.logErrors, "log-errors", false, "Whether to log errors")
	flags.StringVarP(&options.query, "query", "q", "", "The query to open in the web UI bug list")
	flags.StringSliceVar(&options.repos, "repo", nil,
		"Additional repository path to serve (repeatable). The repo name is derived from its parent directory (e.g. /a/b/myorg/myrepo -> myorg/myrepo).")
	flags.StringVar(&options.root, "root", "",
		"Scan this directory for <org>/<repo>/.git siblings and register each as a repo. Multi-repo mode forces --read-only.")

	return cmd
}

// setupRoutes builds the router and registers all API and UI routes.
func setupRoutes(env *execenv.Env, opts webUIOptions) (*mux.Router, func() error, error) {
	router := mux.NewRouter()

	mrc := cache.NewMultiRepoCache()

	// Discover repos for multi-repo mode.
	extraRepos, err := discoverExtraRepos(opts)
	if err != nil {
		return nil, nil, err
	}
	multi := len(extraRepos) > 0

	if multi && !opts.readOnly {
		// Authoring across many repos needs per-repo identity selection which
		// the current auth middleware doesn't support. Force read-only.
		env.Err.Println("multi-repo mode: forcing --read-only (per-repo identity not yet supported)")
		opts.readOnly = true
	}

	if !opts.readOnly {
		author, err := identity.GetUserIdentity(env.Repo)
		if err != nil {
			return nil, nil, err
		}
		router.Use(auth.Middleware(author.Id()))
	}

	// In multi-repo mode, register the cwd repo under its derived org/repo
	// name so it appears in the landing list alongside the scanned siblings.
	// In single-repo mode, keep the existing RegisterDefaultRepository
	// behaviour — no name, served at /.
	cwdPath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	cwdAbs, _ := filepath.Abs(cwdPath)

	var cwdName string
	if multi {
		cwdName = filepath.Base(filepath.Dir(cwdAbs)) + "/" + filepath.Base(cwdAbs)
		env.Out.Printf("  + %s (cwd)\n", cwdName)
		_, events := mrc.RegisterRepository(env.Repo, cwdName)
		if err := execenv.CacheBuildProgressBar(env, events); err != nil {
			return nil, nil, err
		}
	} else {
		_, events := mrc.RegisterDefaultRepository(env.Repo)
		if err := execenv.CacheBuildProgressBar(env, events); err != nil {
			return nil, nil, err
		}
	}

	// Register each extra repo by its derived name. Skip the cwd path if it
	// showed up in the --root scan — the RepoCache lockfile would clash.
	for _, er := range extraRepos {
		if er.path == cwdAbs {
			continue
		}
		r, err := repository.OpenGoGitRepo(er.path, "git-bug", nil)
		if err != nil {
			env.Err.Printf("skipping %s: %v\n", er.path, err)
			continue
		}
		env.Out.Printf("  + %s\n", er.name)
		_, events := mrc.RegisterRepository(r, er.name)
		if err := execenv.CacheBuildProgressBar(env, events); err != nil {
			return nil, nil, err
		}
	}

	var errOut io.Writer
	if opts.logErrors {
		errOut = env.Err
	}

	// Header middleware: when X-Repo-Name is set on an incoming request, the
	// rootQueryResolver.Repository fallback uses it to resolve the default
	// repo instead of erroring on "not unique".
	router.Use(repoNameHeaderMiddleware())

	router.Path("/playground").Handler(playground.Handler("git-bug", "/graphql"))
	router.Path("/graphql").Handler(graphql.NewHandler(mrc, errOut))
	router.Path("/gitfile/{repo}/{rest:.+}").Handler(httpapi.NewGitFileHandler(mrc))
	router.Path("/upload/{repo}").Methods("POST").Handler(httpapi.NewGitUploadFileHandler(mrc))
	router.PathPrefix("/").Handler(webui.NewHandler())

	return router, mrc.Close, nil
}

// extraRepo is a (name, path) pair discovered for multi-repo mode.
type extraRepo struct {
	name string
	path string
}

// discoverExtraRepos resolves --repo and --root flags into a list of repos to
// register beyond the cwd repo. A repo is only included if it has a git-bug
// namespace present (refs/bugs/*).
func discoverExtraRepos(opts webUIOptions) ([]extraRepo, error) {
	var out []extraRepo
	seen := make(map[string]bool)

	add := func(path string) {
		abs, err := filepath.Abs(path)
		if err != nil || seen[abs] {
			return
		}
		seen[abs] = true
		// Skip hidden dirs at either the org or repo level — ".github",
		// ".DS_Store", ".hidden-org", etc. never contain a user-facing repo.
		if strings.HasPrefix(filepath.Base(abs), ".") ||
			strings.HasPrefix(filepath.Base(filepath.Dir(abs)), ".") {
			return
		}
		if !hasGitBugData(abs) {
			return
		}
		// Name is the parent-dir-basename + basename (<owner>/<repo>).
		name := filepath.Base(filepath.Dir(abs)) + "/" + filepath.Base(abs)
		out = append(out, extraRepo{name: name, path: abs})
	}

	for _, p := range opts.repos {
		add(p)
	}
	if opts.root != "" {
		matches, err := filepath.Glob(filepath.Join(opts.root, "*", "*"))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if info, statErr := os.Stat(m); statErr == nil && info.IsDir() {
				add(m)
			}
		}
	}
	return out, nil
}

// hasGitBugData reports whether the repo at path has any git-bug refs. Used
// to skip sibling dirs that happen to be git repos but aren't synced.
func hasGitBugData(path string) bool {
	for _, p := range []string{
		filepath.Join(path, ".git", "refs", "bugs"),
		filepath.Join(path, ".git", "packed-refs"),
	} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// repoNameHeaderMiddleware extracts X-Repo-Name from incoming HTTP requests
// and stashes it in the request context so the GraphQL layer can use it as
// the default repo when no explicit ref is supplied.
func repoNameHeaderMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := r.Header.Get("X-Repo-Name")
			if name != "" {
				ctx := repoctx.WithName(r.Context(), name)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func runWebUI(env *execenv.Env, opts webUIOptions) error {
	router, closeRoutes, err := setupRoutes(env, opts)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeRoutes(); err != nil {
			env.Err.Println(err)
		}
	}()

	if opts.port == 0 {
		opts.port, err = freeport.GetFreePort()
		if err != nil {
			return err
		}
	}

	addr := net.JoinHostPort(opts.bind, strconv.Itoa(opts.port))
	server := &http.Server{Addr: addr, Handler: router}
	baseURL := "http://" + addr

	env.Out.Printf("Web UI: %s\n", baseURL)
	env.Out.Printf("Graphql API: %s/graphql\n", baseURL)
	env.Out.Printf("Graphql Playground: %s/playground\n", baseURL)
	env.Out.Printf("\n[ Press Ctrl+c to quit ]\n\n")

	toOpen := baseURL
	if len(opts.query) > 0 {
		toOpen = fmt.Sprintf("%s/?q=%s", baseURL, url.QueryEscape(opts.query))
	}
	configOpen, err := env.Repo.AnyConfig().ReadBool(webUIOpenConfigKey)
	if errors.Is(err, repository.ErrNoConfigEntry) {
		// default to true
		configOpen = true
	} else if err != nil {
		return err
	}
	if (configOpen && !opts.noOpen) || opts.open {
		go openWhenUp(env, toOpen)
	}

	go func() {
		<-env.Ctx.Done()
		env.Out.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(shutdownCtx); err != nil {
			env.Err.Printf("Could not gracefully shutdown the HTTP server: %v\n", err)
		}
	}()

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func openWhenUp(env *execenv.Env, toOpen string) {
	const maxAttempts = 3
	if isUp(toOpen, maxAttempts, 3*time.Second) {
		if err := open.Run(toOpen); err != nil {
			env.Err.Println(err)
			return
		}
		env.Out.Printf("opened your default browser to url: %s\n", toOpen)
		return
	}
	env.Err.Printf(
		"uh oh! it appears that the http server hasn't started.\n"+
			"we failed to reach %s after %d attempts.\n",
		toOpen, maxAttempts,
	)
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
