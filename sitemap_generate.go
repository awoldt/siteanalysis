package main

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

var skipPrefixes = []string{
	"/cdn-cgi/",
	"/wp-admin/",
	"/wp-login",
	"/wp-json/",
	"/wp-includes/",
	"/xmlrpc.php",
	"/cgi-bin/",
	"/.well-known/",
	"/feed/",
	"/comments/feed/",
	"/author/",
	"/tag/",
	"/category/",
	"/search/",
	"/wp-content/plugins/",
	"/wp-content/themes/",
}

func collectSiteLinks(url *url.URL) ([]string, error) {
	// this function will parse the entirety of a html page
	// only to find all the <a> tags

	// it will loop until it finds all <a> tags across all
	// unique pages

	basePath := fmt.Sprintf("%v://%v", url.Scheme, url.Host)

	var uniquePages []string = []string{url.String()}
	var urlToFetch = url.String()
	var maxPageScans = 5

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
					link := attr.Val

					if attr.Key == "href" && link != "" {
						println(link)

						// we only want relative urls
						if link[0] != '/' {
							continue
						}

						invalidPrefix := false
						// make sure not some stupid injected link
						for _, v := range skipPrefixes {
							if strings.HasPrefix(link, v) {
								invalidPrefix = true
								break
							}
						}
						if invalidPrefix {
							continue
						}

						s := basePath + link

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
