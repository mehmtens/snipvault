package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFiles embed.FS

func registerUI(mux *http.ServeMux) {
	assets, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(assets))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", fileServer))
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		serveWebFile(w, "web/docs.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		serveWebFile(w, "web/openapi.yaml", "application/yaml; charset=utf-8")
	})
	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("GET /p/{slug}", serveIndex)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	serveWebFile(w, "web/index.html", "text/html; charset=utf-8")
}

func serveWebFile(w http.ResponseWriter, name, contentType string) {
	data, err := webFiles.ReadFile(name)
	if err != nil {
		http.Error(w, "page could not be loaded", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}
