package main

import (
	"encoding/json/v2"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// home (html page)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("index.html")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("error while loading page :("))
			return
		}

		w.Header().Set("content-type", "text/html")
		w.Write(file)
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("robots.txt")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("error while loading page :("))
			return
		}

		w.Header().Set("content-type", "text/plain")
		w.Write(file)
	})

	r.Get("/favicon-96x96.png", serveFile("assets/favicon-96x96.png"))
	r.Get("/favicon.svg", serveFile("assets/favicon.svg"))
	r.Get("/favicon.ico", serveFile("assets/favicon.ico"))
	r.Get("/apple-touch-icon.png", serveFile("assets/apple-touch-icon.png"))
	r.Get("/web-app-manifest-192x192.png", serveFile("assets/web-app-manifest-192x192.png"))
	r.Get("/web-app-manifest-512x512.png", serveFile("assets/web-app-manifest-512x512.png"))

	r.Get("/api/html", func(w http.ResponseWriter, r *http.Request) {
		validUrl, err := extractAbsoluteSiteQuery(r.RequestURI)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		siteResponse, err := fetchSite(validUrl)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		data, err := json.Marshal(siteResponse)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("content-type", "application/json")
		w.Write(data)
	})

	http.ListenAndServe(":8080", r)
}
