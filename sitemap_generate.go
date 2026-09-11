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

func collectPageLinks(url *url.URL) ([]string, error) {
	// this function will parse the entirety of a html page
	// only to find all the <a> tags

	basePath := getRootUrl(url)

	var uniqueLinks []string

	req, err := http.NewRequest("GET", url.String(), nil)
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
					// we only want relative urls
					if link[0] != '/' {
						continue
					}

					invalidPrefix := false
					// make sure not some stupid injected links
					// must be a link that is originally placed by the site
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
					if !slices.Contains(uniqueLinks, s) {
						uniqueLinks = append(uniqueLinks, s)
					}
				}
			}
		}
	}

	return uniqueLinks, nil
}

func getRootUrl(urlStr *url.URL) string {
	// gets the root of a site url
	// ex: https://awoldt.dev/articles/not-using-ai-made-me-happy-again -> https://awoldt.dev

	return fmt.Sprintf("%v://%v", urlStr.Scheme, urlStr.Host)
}

func generateSitemapString(links []string) string {
	var str strings.Builder

	str.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	str.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">")

	for _, v := range links {
		str.WriteString("<url>")
		str.WriteString(fmt.Sprintf("<loc>%v</loc>", v))
		str.WriteString("</url>")
	}

	str.WriteString("</urlset>")
	return str.String()
}
