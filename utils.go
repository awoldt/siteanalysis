package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type HeaderTag = map[string][]string

type ScriptTag struct {
	Src     *string `json:"src,omitempty"`
	Content *string `json:"content,omitempty"`
}

type HtmlDetails struct {
	Title         *string      `json:"title"`
	Description   *string      `json:"description"`
	HeaderTags    *HeaderTag   `json:"headerTags"`
	SpanTags      *[]string    `json:"spanTags"`
	ImgTags       *[]string    `json:"imageTags"`
	ParagraphTags *[]string    `json:"paragraphTags"`
	ScriptTags    *[]ScriptTag `json:"scriptTags"`
}

type SiteResponse struct {
	Url           string      `json:"url"`
	UrlQuery      *url.Values `json:"urlQuery"`
	Status        string      `json:"status"`
	StatusCode    int         `json:"statusCode"`
	ResponseTime  string      `json:"responseTime"`
	ContentLength *int64      `json:"contentLength"`
	ContentType   *string     `json:"contentType"`
	HtmlDetails   HtmlDetails `json:"html"`
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

	var urlQuery *url.Values
	if len(urlStr.Query()) > 0 {
		v := urlStr.Query()
		urlQuery = &v
	}

	return SiteResponse{
		Url:           rawUrl,
		UrlQuery:      urlQuery,
		StatusCode:    res.StatusCode,
		Status:        res.Status,
		ResponseTime:  time.Since(startTime).String(),
		ContentLength: contentLength,
		ContentType:   contentType,
		HtmlDetails:   parseHtml(res),
	}, nil
}

func parseHtml(response *http.Response) HtmlDetails {
	html, err := html.Parse(response.Body)
	if err != nil {
		return HtmlDetails{}
	}

	var htmlData HtmlDetails

	// loop through the DOM tree
	for n := range html.Descendants() {
		switch n.Data {

		case "title":
			{
				if n.FirstChild == nil || n.FirstChild.Data == "" {
					continue
				}

				if htmlData.Title == nil || *htmlData.Title == "" {
					s := strings.TrimSpace(n.FirstChild.Data)
					htmlData.Title = &s
				}
			}

		case "meta":
			{
				if len(n.Attr) == 0 || htmlData.Description != nil {
					continue
				}

				// loop through all the attributes on this tag until you find name="description"
				// then loop again until you find the contet="xxxx" attr (the actual description we want)
				b := false
				for _, v := range n.Attr {
					if v.Key == "name" && v.Val == "description" {
						for _, v2 := range n.Attr {
							if v2.Key == "content" {
								s := strings.TrimSpace(v2.Val)
								htmlData.Description = &s
								b = true
								break
							}
						}
					}
					if b {
						break
					}
				}
			}

		case "span":
			{
				if n.FirstChild == nil || n.FirstChild.Data == "" {
					continue
				}

				s := strings.TrimSpace(n.FirstChild.Data)

				if htmlData.SpanTags == nil {
					htmlData.SpanTags = &[]string{s}
				} else {
					*htmlData.SpanTags = append(*htmlData.SpanTags, s)
				}
			}

		case "p":
			{
				if n.FirstChild == nil || n.FirstChild.Data == "" {
					continue
				}

				s := strings.TrimSpace(n.FirstChild.Data)

				if htmlData.ParagraphTags == nil {
					htmlData.ParagraphTags = &[]string{s}
				} else {
					*htmlData.ParagraphTags = append(*htmlData.ParagraphTags, s)
				}
			}

		case "img":
			{
				if len(n.Attr) == 0 {
					continue
				}

				// loop through all the attributes on this tag until you find src
				for _, v := range n.Attr {
					if v.Key == "src" && v.Val != "" {
						if htmlData.ImgTags == nil {
							htmlData.ImgTags = &[]string{v.Val}
						} else {
							*htmlData.ImgTags = append(*htmlData.ImgTags, v.Val)
						}

						break
					}
				}
			}

		case "h1", "h2", "h3", "h4", "h5", "h6":
			{
				if n.FirstChild == nil || n.FirstChild.Data == "" {
					continue
				}

				if htmlData.HeaderTags == nil {
					htmlData.HeaderTags = &map[string][]string{}
				}

				s := strings.TrimSpace(n.FirstChild.Data)

				x := (*htmlData.HeaderTags)[n.Data]
				x = append(x, s)
				(*htmlData.HeaderTags)[n.Data] = x
			}

		case "script":
			{
				// determine if this is an inline or external script tag
				scriptSrc := ""
				for _, v := range n.Attr {
					if v.Key == "src" {
						scriptSrc = v.Val
						break
					}
				}

				if scriptSrc == "" {
					// inline
					if n.FirstChild != nil && n.FirstChild.Data != "" {
						if htmlData.ScriptTags == nil {
							htmlData.ScriptTags = &[]ScriptTag{{Content: &n.FirstChild.Data}}
						} else {
							*htmlData.ScriptTags = append(*htmlData.ScriptTags, ScriptTag{Content: &n.FirstChild.Data})
						}
					}
				} else {
					// external
					// determine if src to script is from site origin or external sit

					url, _ := url.ParseRequestURI(scriptSrc)
					if url.Host == "" {
						// points to origin site
						scriptSrc = fmt.Sprintf("%v://%v%v", response.Request.URL.Scheme, response.Request.Host, scriptSrc)
					} else {
						// points to external site
						scriptSrc = fmt.Sprintf("%v://%v%v", url.Scheme, url.Host, url.Path)
					}

					if htmlData.ScriptTags == nil {
						htmlData.ScriptTags = &[]ScriptTag{{Src: &scriptSrc}}
					} else {
						*htmlData.ScriptTags = append(*htmlData.ScriptTags, ScriptTag{Src: &scriptSrc})
					}
				}
			}
		}
	}

	return htmlData
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
	if index == -1 {
		return nil, fmt.Errorf("missing site url query")
	}

	// take the entire string after "site="
	absoluteUrl := urlStr[index+5:]

	// ensure this is a real absolute url
	url, err := url.ParseRequestURI(absoluteUrl)
	if err != nil {
		return nil, fmt.Errorf("site url query is not a valid absolute url")
	}

	return url, nil
}
