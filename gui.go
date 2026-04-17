package main

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed frontend/dist/* frontend/dist/assets/*
var embeddedFrontendDist embed.FS

var (
	frontendDistFS     fs.FS
	frontendDistServer http.Handler
)

func init() {
	var err error
	frontendDistFS, err = fs.Sub(embeddedFrontendDist, "frontend/dist")
	if err != nil {
		panic(err)
	}
	frontendDistServer = http.FileServer(http.FS(frontendDistFS))
}

func serveEmbeddedFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if requestPath == "." || requestPath == "" {
		requestPath = "index.html"
	}

	if strings.HasPrefix(requestPath, "api/") {
		http.NotFound(w, r)
		return
	}

	if fileInfo, err := fs.Stat(frontendDistFS, requestPath); err == nil && !fileInfo.IsDir() {
		frontendDistServer.ServeHTTP(w, r)
		return
	}

	indexContents, err := fs.ReadFile(frontendDistFS, "index.html")
	if err != nil {
		http.Error(w, "frontend index not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(indexContents))
}
