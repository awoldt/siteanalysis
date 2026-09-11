package main

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"golang.org/x/net/html"
)

func collectSiteLinks(url *url.URL) ([]string, error) {
	// this function will parse the entirety of a html page
	// only to find all the <a> tags

	// it will loop until it finds all <a> tags across all
	// unique pages

	basePath := fmt.Sprintf("%v://%v", url.Scheme, url.Host)

	var uniquePages []string = []string{url.String()}
	var urlToFetch = url.String()
	var maxPageScans = 25

	// we will loop and parse the entire site across all pages
	// until all the unique links have been collected
	i := 0
	for {
		if i > maxPageScans {
			break
		}

		req, err := http.NewRequest("GET", urlToFetch, nil)
		if err != nil {
			return nil, err
		}
		// set a customer header to prevent some sites from throwing 403s
		req.Header.Set("User-Agent", CUSTOM_USER_AGENT)

		res, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		html, err := html.Parse(res.Body)
		if err != nil {
			return nil, err
		}

		for n := range html.Descendants() {
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" && attr.Val != "" {
						// we only want relative urls
						if attr.Val[0] != '/' {
							continue
						}

						s := basePath + attr.Val

						// make sure we have not already added this link
						if !slices.Contains(uniquePages, s) {
							uniquePages = append(uniquePages, s)
						}
					}
				}
			}
		}
		i++
	}

	return uniquePages, nil
}
