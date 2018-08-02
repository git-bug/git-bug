package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MichaelMure/git-bug/graphql"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"github.com/MichaelMure/git-bug/webui"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlgen/handler"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var port int

func runWebUI(cmd *cobra.Command, args []string) error {
	if port == 0 {
		var err error
		port, err = freeport.GetFreePort()
		if err != nil {
			return err
		}
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	webUiAddr := fmt.Sprintf("http://%s", addr)

	fmt.Printf("Web UI: %s\n", webUiAddr)
	fmt.Printf("Graphql API: http://%s/graphql\n", addr)
	fmt.Printf("Graphql Playground: http://%s/playground\n", addr)

	router := mux.NewRouter()

	// Routes
	router.Path("/playground").Handler(handler.Playground("git-bug", "/graphql"))
	router.Path("/graphql").Handler(graphql.NewHandler(repo))
	router.Path("/gitfile/{hash}").Handler(newGitFileHandler(repo))
	router.Path("/upload").Methods("POST").Handler(newGitUploadFileHandler(repo))
	router.PathPrefix("/").Handler(http.FileServer(webui.WebUIAssets))

	open.Run(webUiAddr)

	log.Fatal(http.ListenAndServe(addr, router))

	return nil
}

type gitFileHandler struct {
	repo repository.Repo
}

func newGitFileHandler(repo repository.Repo) http.Handler {
	return &gitFileHandler{
		repo: repo,
	}
}

func (gfh *gitFileHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	hash := mux.Vars(r)["hash"]

	if !isGitHash(hash) {
		http.Error(rw, "invalid git hash", http.StatusBadRequest)
		return
	}

	// TODO: this mean that the whole file will he buffered in memory
	// This can be a problem for big files. There might be a way around
	// that by implementing a io.ReadSeeker that would read and discard
	// data when a seek is called.
	data, err := gfh.repo.ReadData(util.Hash(hash))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeContent(rw, r, "", time.Now(), bytes.NewReader(data))
}

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
	rw.Write(js)
}

func isGitHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

var webUICmd = &cobra.Command{
	Use:   "webui",
	Short: "Launch the web UI",
	RunE:  runWebUI,
}

func init() {
	RootCmd.AddCommand(webUICmd)
	webUICmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen to")
}
