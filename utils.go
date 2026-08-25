package main

import (
	"io"
	"net/http"
	"time"

	"golang.org/x/net/html"
)

type HtmlDetails struct {
	Title       string
	Description string
}

type SiteResponse struct {
	StatusCode    int          `json:"statusCode"`
	ResponseTime  string       `json:"responseTime"`
	ContentLength *int64       `json:"contentLength"`
	ContentType   *string      `json:"contentType"`
	HtmlDetails   *HtmlDetails `json:"html"`
}

func fetchSite(url string) (SiteResponse, error) {
	startTime := time.Now()

	res, err := http.Get(url)
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
		StatusCode:    res.StatusCode,
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
