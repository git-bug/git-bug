package webui

import (
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

func init() {
	// Override OS-specific MIME registry (Windows maps .js → application/javascript).
	mime.AddExtensionType(".js", "text/javascript")
	mime.AddExtensionType(".mjs", "text/javascript")
	mime.AddExtensionType(".css", "text/css")
}

// NewHandler returns an http.Handler that serves the webui SPA.
// Extension-less paths fall back to index.html for client-side routing.
// Pre-compressed .gz files are served directly when the client supports gzip;
// otherwise they are decompressed on the fly.
func NewHandler() http.Handler {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return &spaHandler{http.FS(dist)}
}

type spaHandler struct {
	fs http.FileSystem
}

type resolvedFile struct {
	file        http.File
	logicalPath string // original path, used for Content-Type
	sendAsGzip  bool   // true → set Content-Encoding: gzip
	decompress  bool   // true → decompress on the fly
}

// resolve opens the best available file for logicalPath.
// It prefers .gz when the client accepts gzip, falls back to plain,
// and as a last resort opens .gz for on-the-fly decompression.
func (s *spaHandler) resolve(logicalPath string, acceptsGzip bool) (*resolvedFile, error) {
	if acceptsGzip {
		if f, err := s.openFile(logicalPath + ".gz"); err == nil {
			return &resolvedFile{file: f, logicalPath: logicalPath, sendAsGzip: true}, nil
		}
	}
	// data is already decompressed
	if f, err := s.openFile(logicalPath); err == nil {
		return &resolvedFile{file: f, logicalPath: logicalPath}, nil
	}
	// Client doesn't accept gzip but only .gz exists: decompress on the fly.
	if f, err := s.openFile(logicalPath + ".gz"); err == nil {
		return &resolvedFile{file: f, logicalPath: logicalPath, decompress: true}, nil
	}
	return nil, fs.ErrNotExist
}

func (s *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if urlPath == "" || urlPath == "/" {
		urlPath = "/index.html"
	}

	acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

	rf, err := s.resolve(urlPath, acceptsGzip)
	if err != nil {
		// Only fall back to index.html for extension-less paths (SPA routes).
		if filepath.Ext(urlPath) != "" {
			http.NotFound(w, r)
			return
		}
		rf, err = s.resolve("/index.html", acceptsGzip)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	defer rf.file.Close()

	ct := mime.TypeByExtension(filepath.Ext(rf.logicalPath))
	if ct == "" {
		ct = "application/octet-stream"
	}

	stat, err := rf.file.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	switch {
	case rf.sendAsGzip:
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		http.ServeContent(w, r, rf.logicalPath, stat.ModTime(), rf.file)

	case rf.decompress:
		gr, err := gzip.NewReader(rf.file)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer gr.Close()
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, gr)

	default:
		http.ServeContent(w, r, rf.logicalPath, stat.ModTime(), rf.file)
	}
}

// openFile opens a non-directory file, returning an error for directories.
func (s *spaHandler) openFile(name string) (http.File, error) {
	f, err := s.fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		_ = f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}
