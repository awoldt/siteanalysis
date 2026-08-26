package main

import (
	"encoding/json/v2"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
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
