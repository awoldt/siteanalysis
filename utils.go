package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type HtmlDetails struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type SiteResponse struct {
	Url           string       `json:"url"`
	UrlQuery      url.Values   `json:"urlQuery"`
	Status        string       `json:"status"`
	StatusCode    int          `json:"statusCode"`
	ResponseTime  string       `json:"responseTime"`
	ContentLength *int64       `json:"contentLength"`
	ContentType   *string      `json:"contentType"`
	HtmlDetails   *HtmlDetails `json:"html"`
}

func fetchSite(urlStr *url.URL) (SiteResponse, error) {
	startTime := time.Now()
	rawUrl := urlStr.String()

	res, err := http.Get(rawUrl)
	if err != nil {
		return SiteResponse{}, err
	}

	defer res.Body.Close()

	var contentLength *int64
	if res.ContentLength != -1 {
		contentLength = &res.ContentLength
	}

	var contentType *string
	ct := res.Header.Get("Content-Type")
	if ct != "" {
		contentType = &ct
	}

	return SiteResponse{
		Url:           rawUrl,
		UrlQuery:      urlStr.Query(),
		StatusCode:    res.StatusCode,
		Status:        res.Status,
		ResponseTime:  time.Since(startTime).String(),
		ContentLength: contentLength,
		ContentType:   contentType,
		HtmlDetails:   parseHtml(res.Body),
	}, nil
}

func parseHtml(body io.Reader) *HtmlDetails {
	html, err := html.Parse(body)
	if err != nil {
		return nil
	}

	var title string
	var description string

	// loop through the DOM tree
	for n := range html.Descendants() {
		if title != "" && description != "" {
			break
		}

		// title
		if n.Data == "title" && n.FirstChild != nil && title == "" {
			title = n.FirstChild.Data
		}

		// description
		if n.Data == "meta" && len(n.Attr) > 0 {
			// loop through all the attributes on this tag until you find name="description"
			// then loop again until you find the contet="xxxx" attr (the actual description we want)
			for _, v := range n.Attr {
				if description != "" {
					break
				}

				if v.Key == "name" && v.Val == "description" {
					for _, v2 := range n.Attr {
						if v2.Key == "content" {
							description = v2.Val
							break
						}
					}
				}
			}

		}
	}

	return &HtmlDetails{Title: title, Description: description}
}

func extractAbsoluteSiteQuery(urlStr string) (*url.URL, error) {
	// we first need to extract the site url query param
	// from the url
	// this is slighly more involved bc the site query value
	// is an actual absolute url... which can also contain more ? in the url

	// example: "?site=https://google.com?search=hello&device=phone"
	// the "device" url query for the site url query wont be picked up

	// find the index of "site="
	index := strings.Index(urlStr, "site=")

	// take the entire string after "site="
	absoluteUrl := urlStr[index+5:]

	// ensure this is a real absolute url
	url, err := url.ParseRequestURI(absoluteUrl)
	if err != nil {
		return nil, fmt.Errorf("site url query is not a valid absolute url")
	}

	return url, nil
}
