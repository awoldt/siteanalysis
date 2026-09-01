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

type ListTags struct {
	OrderedLists   *[]string `json:"orderedList,omitempty"`
	UnorderedLists *[]string `json:"unorderedLists,omitempty"`
}

type HtmlDetails struct {
	Title         *string      `json:"title"`
	Description   *string      `json:"description"`
	HeaderTags    *HeaderTag   `json:"headerTags"`
	SpanTags      *[]string    `json:"spanTags"`
	ImgTags       *[]string    `json:"imageTags"`
	ParagraphTags *[]string    `json:"paragraphTags"`
	ScriptTags    *[]ScriptTag `json:"scriptTags"`
	AnchorTags    *[]string    `json:"anchorTags"`
	Lists         *ListTags    `json:"listTags"`
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
				text := extractText(n)
				if text == "" {
					continue
				}

				if htmlData.Title == nil || *htmlData.Title == "" {
					htmlData.Title = &text
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
				text := extractText(n)
				if text == "" {
					continue
				}

				if htmlData.SpanTags == nil {
					htmlData.SpanTags = &[]string{text}
				} else {
					*htmlData.SpanTags = append(*htmlData.SpanTags, text)
				}
			}

		case "p":
			{
				text := extractText(n)
				if text == "" {
					continue
				}

				if htmlData.ParagraphTags == nil {
					htmlData.ParagraphTags = &[]string{text}
				} else {
					*htmlData.ParagraphTags = append(*htmlData.ParagraphTags, text)
				}
			}

		case "img":
			{
				if len(n.Attr) == 0 {
					continue
				}

				src := ""

				// loop through all the attributes on this tag until you find src
				for _, v := range n.Attr {
					if v.Key == "src" && v.Val != "" {
						src = getSrcUrl(v.Val, response)
						if src == "" {
							continue
						}

						if htmlData.ImgTags == nil {
							htmlData.ImgTags = &[]string{src}
						} else {
							*htmlData.ImgTags = append(*htmlData.ImgTags, src)
						}

						break
					}
				}
			}

		case "h1", "h2", "h3", "h4", "h5", "h6":
			{
				text := extractText(n)
				if text == "" {
					continue
				}

				if htmlData.HeaderTags == nil {
					htmlData.HeaderTags = &map[string][]string{}
				}

				x := (*htmlData.HeaderTags)[n.Data]
				x = append(x, text)
				(*htmlData.HeaderTags)[n.Data] = x
			}

		case "script":
			{
				// determine if this is an inline or external script tag
				scriptSrc := ""
				for _, v := range n.Attr {
					if v.Key == "src" && v.Val != "" {
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
					scriptSrc = getSrcUrl(scriptSrc, response)
					if scriptSrc == "" {
						continue
					}

					if htmlData.ScriptTags == nil {
						htmlData.ScriptTags = &[]ScriptTag{{Src: &scriptSrc}}
					} else {
						*htmlData.ScriptTags = append(*htmlData.ScriptTags, ScriptTag{Src: &scriptSrc})
					}
				}
			}

		case "a":
			{
				if len(n.Attr) == 0 {
					continue
				}

				href := ""
				for _, v := range n.Attr {
					if v.Key == "href" && v.Val != "" {
						href = getSrcUrl(v.Val, response)
						if href == "" {
							continue
						}

						break
					}
				}

				if htmlData.AnchorTags == nil {
					htmlData.AnchorTags = &[]string{href}
				} else {
					*htmlData.AnchorTags = append(*htmlData.AnchorTags, href)
				}
			}

		case "ul", "ol":
			{
				if n.FirstChild == nil || n.FirstChild.Data == "" {
					continue
				}

				// loop through all the lis of the list
				for li := range n.Descendants() {
					if li.Data != "li" || li.FirstChild == nil || li.FirstChild.Data == "" {
						continue
					}

					switch n.Data {
					case "ol":
						{
							if htmlData.Lists == nil || htmlData.Lists.OrderedLists == nil {
								htmlData.Lists = &ListTags{OrderedLists: &[]string{li.FirstChild.Data}}
							} else {
								x := htmlData.Lists
								*x.OrderedLists = append(*x.OrderedLists, li.FirstChild.Data)
								htmlData.Lists = x
							}
						}

					case "ul":
						{
							if htmlData.Lists == nil || htmlData.Lists.UnorderedLists == nil {
								htmlData.Lists = &ListTags{UnorderedLists: &[]string{li.FirstChild.Data}}
							} else {
								x := htmlData.Lists
								*x.UnorderedLists = append(*x.UnorderedLists, li.FirstChild.Data)
								htmlData.Lists = x
							}
						}
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

func getSrcUrl(str string, response *http.Response) string {
	// this function will take in either an absolute or relative url
	// and return an absolute
	// its used for getting think like img src and script src urls...

	// for example, if i parse https://google.com and this has a script tag that
	// loads js from a differnet domain, the result for the script tag src would be
	// relative "/js/data.js".. when we need it to be "https://othersite.com/js/data.js" instead

	src := ""

	url, err := url.ParseRequestURI(str)
	if err != nil {
		return ""
	}

	if url.Host == "" {
		// points to origin site
		src = fmt.Sprintf("%v://%v%v", response.Request.URL.Scheme, response.Request.Host, str)
	} else {
		// points to external site
		src = fmt.Sprintf("%v://%v%v", url.Scheme, url.Host, url.Path)
	}

	return src
}

func extractText(n *html.Node) string {
	// walk down the dom tree until you find the inner-most child
	// with a type of "text" node

	// more sibling to sibling across nodes and collect all text

	// ex: the h1 tag below should extract "Business openings and closings in August 2026 roundup"
	// <h1>
	//   Business <span>openings and <a href="#">closings</a></span> in
	//   <em>August</em>
	//   <span><b>2026</b> roundup</span>
	// </h1>

	var text strings.Builder

	for node := range n.Descendants() {
		data := strings.TrimSpace(node.Data)

		if data == "" {
			continue
		}

		if node.Type == html.TextNode {
			text.WriteString(data)
		}
	}

	return text.String()
}
