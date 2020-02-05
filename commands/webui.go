package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	"github.com/MichaelMure/git-bug/graphql"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/webui"
)

var (
	webUIPort   int
	webUIOpen   bool
	webUINoOpen bool
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

	graphqlHandler, err := graphql.NewHandler(repo)
	if err != nil {
		return err
	}

	assetsHandler := &fileSystemWithDefault{
		FileSystem:  webui.WebUIAssets,
		defaultFile: "index.html",
	}

	// Routes
	router.Path("/playground").Handler(playground.Handler("git-bug", "/graphql"))
	router.Path("/graphql").Handler(graphqlHandler)
	router.Path("/gitfile/{hash}").Handler(newGitFileHandler(repo))
	router.Path("/upload").Methods("POST").Handler(newGitUploadFileHandler(repo))
	router.PathPrefix("/").Handler(http.FileServer(assetsHandler))

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

// implement a http.Handler that will read and server git blob.
type gitFileHandler struct {
	repo repository.Repo
}

func newGitFileHandler(repo repository.Repo) http.Handler {
	return &gitFileHandler{
		repo: repo,
	}
}

func (gfh *gitFileHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	hash := git.Hash(mux.Vars(r)["hash"])

	if !hash.IsValid() {
		http.Error(rw, "invalid git hash", http.StatusBadRequest)
		return
	}

	// TODO: this mean that the whole file will he buffered in memory
	// This can be a problem for big files. There might be a way around
	// that by implementing a io.ReadSeeker that would read and discard
	// data when a seek is called.
	data, err := gfh.repo.ReadData(git.Hash(hash))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeContent(rw, r, "", time.Now(), bytes.NewReader(data))
}

// implement a http.Handler that will accept and store content into git blob.
type gitUploadFileHandler struct {
	repo repository.Repo
}

func newGitUploadFileHandler(repo repository.Repo) http.Handler {
	return &gitUploadFileHandler{
		repo: repo,
	}
}

func (gufh *gitUploadFileHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// 100MB (github limit)
	var maxUploadSize int64 = 100 * 1000 * 1000
	r.Body = http.MaxBytesReader(rw, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(rw, "file too big (100MB max)", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("uploadfile")
	if err != nil {
		http.Error(rw, "invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(rw, "invalid file", http.StatusBadRequest)
		return
	}

	filetype := http.DetectContentType(fileBytes)
	if filetype != "image/jpeg" && filetype != "image/jpg" &&
		filetype != "image/gif" && filetype != "image/png" {
		http.Error(rw, "invalid file type", http.StatusBadRequest)
		return
	}

	hash, err := gufh.repo.StoreData(fileBytes)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	type response struct {
		Hash string `json:"hash"`
	}

	resp := response{Hash: string(hash)}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(js)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

var webUICmd = &cobra.Command{
	Use:   "webui",
	Short: "Launch the web UI.",
	Long: `Launch the web UI.

Available git config:
  git-bug.webui.open [bool]: control the automatic opening of the web UI in the default browser
`,
	PreRunE: loadRepoEnsureUser,
	RunE:    runWebUI,
}

func init() {
	RootCmd.AddCommand(webUICmd)

	webUICmd.Flags().SortFlags = false

	webUICmd.Flags().BoolVar(&webUIOpen, "open", false, "Automatically open the web UI in the default browser")
	webUICmd.Flags().BoolVar(&webUINoOpen, "no-open", false, "Prevent the automatic opening of the web UI in the default browser")
	webUICmd.Flags().IntVarP(&webUIPort, "port", "p", 0, "Port to listen to (default is random)")

}
