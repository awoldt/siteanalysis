package main

import (
	"encoding/json/v2"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		siteUrl := queryParams.Get("site")
		if siteUrl == "" {
			w.WriteHeader(400)
			w.Write([]byte("missing site url query"))
			return
		}

		// make sure this is a legit http url
		_, err := url.ParseRequestURI(siteUrl)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("not a valid site url"))
			return
		}

		siteResponse, err := fetchSite(siteUrl, r.Body)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("content-type", "application/json")

		data, err := json.Marshal(siteResponse)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(200)
		w.Write(data)
	})

	http.ListenAndServe("127.0.0.1:3000", r)
}
