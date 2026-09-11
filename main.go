package main

import (
	"encoding/json/v2"
	"net/http"
	"os"
	"slices"

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

	r.Get("/favicon-96x96.png", serveFile("favicon-96x96.png"))
	r.Get("/favicon.svg", serveFile("favicon.svg"))
	r.Get("/favicon.ico", serveFile("favicon.ico"))
	r.Get("/apple-touch-icon.png", serveFile("apple-touch-icon.png"))
	r.Get("/site.webmanifest", serveFile("site.webmanifest"))
	r.Get("/web-app-manifest-192x192.png", serveFile("web-app-manifest-192x192.png"))
	r.Get("/web-app-manifest-512x512.png", serveFile("web-app-manifest-512x512.png"))

	r.Get("/api", func(w http.ResponseWriter, r *http.Request) {
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

	r.Get("/api/sitemap", func(w http.ResponseWriter, r *http.Request) {
		validUrl, err := extractAbsoluteSiteQuery(r.RequestURI)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		// make sure we start scanning at the ROOT of the url passed
		rootUrl := getRootUrl(validUrl)

		// recursively scan each page in the site and return
		// as many unique internal links as possible
		collectedLinks := []string{rootUrl}
		scannedUrls := []string{}
		urlToScan := rootUrl // start with the url provided in the query param

		i := 0
		maxDepth := 100
		for {
			if i > maxDepth {
				break
			}

			links, err := collectPageLinks(validUrl)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			i++
			scannedUrls = append(scannedUrls, urlToScan)

			// only append links that are NOT already stored
			for _, v := range links {
				if !slices.Contains(collectedLinks, v) {
					collectedLinks = append(collectedLinks, v)
				}
			}

			// after scanning all the links for a page
			// find a url that has not been scanned and collect
			// links from that page next
			b := true
			for _, v := range collectedLinks {
				if !slices.Contains(scannedUrls, v) {
					urlToScan = v
					b = false
					break
				}
			}
			if b {
				break
			}
		}

		sitemapStr := generateSitemapString(collectedLinks)

		w.Header().Add("content-type", "application/xml")
		w.Write([]byte(sitemapStr))
	})

	http.ListenAndServe(":8080", r)
}
