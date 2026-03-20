// git_browse_handler.go implements HTTP handlers for browsing git repository
// content (refs, file tree, blobs, raw files, commit history, commit details,
// and per-file diffs). Registered by commands/webui.go alongside the GraphQL
// and legacy file handlers.
package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

// ── shared helpers ────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// repoFromPath resolves the repository from the {owner} and {repo} mux path
// variables. "_" is the wildcard value: owner is always ignored (single-owner
// for now), and repo "_" resolves to the default repository.
func repoFromPath(mrc *cache.MultiRepoCache, r *http.Request) (*cache.RepoCache, error) {
	repoVar := mux.Vars(r)["repo"]
	if repoVar == "_" {
		return mrc.DefaultRepo()
	}
	return mrc.ResolveRepo(repoVar)
}

// objectTypeString maps a repository.ObjectType to the JSON "type" string
// sent to the frontend.
func objectTypeString(ot repository.ObjectType) string {
	switch ot {
	case repository.Tree:
		return "tree"
	case repository.Symlink:
		return "symlink"
	case repository.Submodule:
		return "submodule"
	default:
		return "blob" // Blob, Executable, Unknown all render as files
	}
}

// browseRepo resolves the repository and asserts it implements RepoBrowse.
func browseRepo(mrc *cache.MultiRepoCache, r *http.Request) (repository.RepoBrowse, error) {
	rc, err := repoFromPath(mrc, r)
	if err != nil {
		return nil, err
	}
	browse, err := rc.BrowseRepo()
	if err != nil {
		return nil, fmt.Errorf("repository does not support code browsing")
	}
	return browse, nil
}

// ── GET /api/repos/{owner}/{repo}/git/refs ────────────────────────────────────

type gitRefsHandler struct{ mrc *cache.MultiRepoCache }

func NewGitRefsHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitRefsHandler{mrc: mrc}
}

type refResponse struct {
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Type      string `json:"type"` // "branch" | "tag"
	Hash      string `json:"hash"`
	IsDefault bool   `json:"isDefault"`
}

func (h *gitRefsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	branches, err := repo.Branches()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tags, err := repo.Tags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	refs := make([]refResponse, 0, len(branches)+len(tags))
	for _, b := range branches {
		refs = append(refs, refResponse{
			Name:      "refs/heads/" + b.Name,
			ShortName: b.Name,
			Type:      "branch",
			Hash:      b.Hash.String(),
			IsDefault: b.IsDefault,
		})
	}
	for _, t := range tags {
		refs = append(refs, refResponse{
			Name:      "refs/tags/" + t.Name,
			ShortName: t.Name,
			Type:      "tag",
			Hash:      t.Hash.String(),
		})
	}

	writeJSON(w, refs)
}

// ── GET /api/repos/{owner}/{repo}/git/trees?ref=&path= ───────────────────────

type gitTreeHandler struct{ mrc *cache.MultiRepoCache }

func NewGitTreeHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitTreeHandler{mrc: mrc}
}

type treeEntryResponse struct {
	Name       string              `json:"name"`
	Type       string              `json:"type"` // "tree" | "blob" | "symlink" | "submodule"
	Hash       string              `json:"hash"`
	LastCommit *commitMetaResponse `json:"lastCommit,omitempty"`
}

func (h *gitTreeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entries, err := repo.TreeAtPath(ref, path)
	if err == repository.ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	lastCommits, _ := repo.LastCommitForEntries(ref, path, names)

	resp := make([]treeEntryResponse, 0, len(entries))
	for _, e := range entries {
		item := treeEntryResponse{
			Name: e.Name,
			Type: objectTypeString(e.ObjectType),
			Hash: e.Hash.String(),
		}
		if cm, ok := lastCommits[e.Name]; ok {
			item.LastCommit = toCommitMetaResponse(cm)
		}
		resp = append(resp, item)
	}

	writeJSON(w, resp)
}

// ── GET /api/repos/{owner}/{repo}/git/blobs?ref=&path= ───────────────────────

type gitBlobHandler struct{ mrc *cache.MultiRepoCache }

func NewGitBlobHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitBlobHandler{mrc: mrc}
}

type blobResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	IsBinary bool   `json:"isBinary"`
}

func (h *gitBlobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	if path == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc, size, err := repo.BlobAtPath(ref, path)
	if err == repository.ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	isBinary := bytes.IndexByte(data, 0) >= 0
	content := ""
	if !isBinary {
		content = string(data)
	}

	writeJSON(w, blobResponse{
		Path:     path,
		Content:  content,
		Size:     size,
		IsBinary: isBinary,
	})
}

// ── GET /api/repos/{owner}/{repo}/git/raw?ref=&path= ─────────────────────────
// Serves the raw file content for download. Both ref and path are query
// parameters so that branch names containing slashes are handled correctly.
type gitRawHandler struct{ mrc *cache.MultiRepoCache }

func NewGitRawHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitRawHandler{mrc: mrc}
}

func (h *gitRawHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	if path == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc, size, err := repo.BlobAtPath(ref, path)
	if err == repository.ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	fileName := path[strings.LastIndex(path, "/")+1:]
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	io.Copy(w, rc) //nolint:errcheck
}

// ── GET /api/repos/{owner}/{repo}/git/commits?ref=&path=&limit=&after= ───────

type gitCommitsHandler struct{ mrc *cache.MultiRepoCache }

func NewGitCommitsHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitCommitsHandler{mrc: mrc}
}

type commitMetaResponse struct {
	Hash        string   `json:"hash"`
	ShortHash   string   `json:"shortHash"`
	Message     string   `json:"message"`
	AuthorName  string   `json:"authorName"`
	AuthorEmail string   `json:"authorEmail"`
	Date        string   `json:"date"` // RFC3339
	Parents     []string `json:"parents"`
}

func toCommitMetaResponse(m repository.CommitMeta) *commitMetaResponse {
	parents := make([]string, len(m.Parents))
	for i, p := range m.Parents {
		parents[i] = p.String()
	}
	return &commitMetaResponse{
		Hash:        m.Hash.String(),
		ShortHash:   m.ShortHash,
		Message:     m.Message,
		AuthorName:  m.AuthorName,
		AuthorEmail: m.AuthorEmail,
		Date:        m.Date.UTC().Format("2006-01-02T15:04:05Z"),
		Parents:     parents,
	}
}

func (h *gitCommitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")
	after := repository.Hash(r.URL.Query().Get("after"))

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	if ref == "" {
		http.Error(w, "missing ref", http.StatusBadRequest)
		return
	}

	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	commits, err := repo.CommitLog(ref, path, limit, after)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]*commitMetaResponse, len(commits))
	for i, c := range commits {
		resp[i] = toCommitMetaResponse(c)
	}
	writeJSON(w, resp)
}

// ── GET /api/repos/{owner}/{repo}/git/commits/{sha} ──────────────────────────

type gitCommitHandler struct{ mrc *cache.MultiRepoCache }

func NewGitCommitHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitCommitHandler{mrc: mrc}
}

type changedFileResponse struct {
	Path    string `json:"path"`
	OldPath string `json:"oldPath,omitempty"`
	Status  string `json:"status"`
}

type commitDetailResponse struct {
	*commitMetaResponse
	FullMessage string                `json:"fullMessage"`
	Files       []changedFileResponse `json:"files"`
}

func (h *gitCommitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sha := mux.Vars(r)["sha"]

	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detail, err := repo.CommitDetail(repository.Hash(sha))
	if err == repository.ErrNotFound {
		http.Error(w, "commit not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	files := make([]changedFileResponse, len(detail.Files))
	for i, f := range detail.Files {
		files[i] = changedFileResponse{Path: f.Path, OldPath: f.OldPath, Status: f.Status}
	}

	writeJSON(w, commitDetailResponse{
		commitMetaResponse: toCommitMetaResponse(detail.CommitMeta),
		FullMessage:        detail.FullMessage,
		Files:              files,
	})
}

// ── GET /api/repos/{owner}/{repo}/git/commits/{sha}/diff?path= ───────────────

type gitCommitDiffHandler struct{ mrc *cache.MultiRepoCache }

func NewGitCommitDiffHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitCommitDiffHandler{mrc: mrc}
}

func (h *gitCommitDiffHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sha := mux.Vars(r)["sha"]
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	repo, err := browseRepo(h.mrc, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fd, err := repo.CommitFileDiff(repository.Hash(sha), filePath)
	if err == repository.ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type diffLineResp struct {
		Type    string `json:"type"`
		Content string `json:"content"`
		OldLine int    `json:"oldLine,omitempty"`
		NewLine int    `json:"newLine,omitempty"`
	}
	type diffHunkResp struct {
		OldStart int            `json:"oldStart"`
		OldLines int            `json:"oldLines"`
		NewStart int            `json:"newStart"`
		NewLines int            `json:"newLines"`
		Lines    []diffLineResp `json:"lines"`
	}
	type fileDiffResp struct {
		Path     string         `json:"path"`
		OldPath  string         `json:"oldPath,omitempty"`
		IsBinary bool           `json:"isBinary"`
		IsNew    bool           `json:"isNew"`
		IsDelete bool           `json:"isDelete"`
		Hunks    []diffHunkResp `json:"hunks"`
	}

	hunks := make([]diffHunkResp, len(fd.Hunks))
	for i, h := range fd.Hunks {
		lines := make([]diffLineResp, len(h.Lines))
		for j, l := range h.Lines {
			lines[j] = diffLineResp{Type: l.Type, Content: l.Content, OldLine: l.OldLine, NewLine: l.NewLine}
		}
		hunks[i] = diffHunkResp{OldStart: h.OldStart, OldLines: h.OldLines, NewStart: h.NewStart, NewLines: h.NewLines, Lines: lines}
	}

	writeJSON(w, fileDiffResp{
		Path:     fd.Path,
		OldPath:  fd.OldPath,
		IsBinary: fd.IsBinary,
		IsNew:    fd.IsNew,
		IsDelete: fd.IsDelete,
		Hunks:    hunks,
	})
}
