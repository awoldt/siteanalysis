package main

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var validOpenGraphProperties = []string{
	"og:title",
	"og:description",
	"og:image",
	"og:url",
	"og:type",
	"og:site_name",
	"og:locale",
	"og:audio",
	"og:video",
}

var validFormChildren = []string{
	"input",
	"label",
	"textarea",
	"button",
}

type HeaderTag = map[string][]string  // key: h1,h2,etc value: all strings for each header
type OpenGraphTag = map[string]string // key: "og:title",etc value: content string value

type ScriptTag struct {
	Src     *string `json:"src,omitempty"`
	Content *string `json:"content,omitempty"`
}

type ListTag struct {
	OrderedLists   *[]string `json:"orderedList"`
	UnorderedLists *[]string `json:"unorderedList"`
}

type TableTag struct {
	Headers *[]string `json:"header"` // the header text of each column
	Rows    *[]string `json:"rows"`
}

type FormChild struct {
	Field       string  `json:"field"`
	Type        *string `json:"type,omitempty"`
	Name        *string `json:"name,omitempty"`
	Id          *string `json:"id,omitempty"`
	Value       *string `json:"value,omitempty"`
	Placeholder *string `json:"placeholder,omitempty"`
	Required    *bool   `json:"required,omitempty"`
	Label       *string `json:"label,omitempty"`
	For         *string `json:"for,omitempty"`
}

type FormTag struct {
	Fields FormChild `json:"fields"`
}

type HtmlDetails struct {
	Title         *string       `json:"title"`
	Description   *string       `json:"description"`
	HeaderTags    *HeaderTag    `json:"headers"`
	SpanTags      *[]string     `json:"spans"`
	ImgTags       *[]string     `json:"images"`
	ParagraphTags *[]string     `json:"paragraphs"`
	ScriptTags    *[]ScriptTag  `json:"scripts"`
	LinkTags      *[]string     `json:"links"`
	Lists         *ListTag      `json:"lists"`
	Tables        *[]TableTag   `json:"tables"`
	OpenGraphTags *OpenGraphTag `json:"openGraph"`
	Forms         *[]FormTag    `json:"forms"`
}

type RedirectDetails struct {
	OriginalUrl    string `json:"originalUrl"`
	Status         string `json:"status"`
	StatusCode     int    `json:"statusCode"`
	DestinationUrl string `json:"destinationUrl"`
}

type SiteResponse struct {
	Url             string           `json:"url"`
	UrlQuery        *url.Values      `json:"urlQuery"`
	Status          string           `json:"status"`
	StatusCode      int              `json:"statusCode"`
	RedirectDetails *RedirectDetails `json:"redirectDetails,omitempty"`
	FetchTime       string           `json:"fetchTime"` // time it took to fetch the external website
	ParseTime       string           `json:"parsetime"` // time it took to parse the html and return a response
	ContentLength   *int64           `json:"contentLength"`
	ContentType     *string          `json:"contentType"`
	HtmlDetails     HtmlDetails      `json:"html"`
}

func fetchSite(urlStr *url.URL) (SiteResponse, error) {
	// this is the single function to call that will create the
	// the payload that will be returned from api

	rawUrl := urlStr.String()

	fetchTime := time.Now()
	res, err := http.Get(rawUrl)
	if err != nil {
		return SiteResponse{}, err
	}
	fetchTimeStr := time.Since(fetchTime).String()

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

	var redirectDetails *RedirectDetails
	if res.Request.Response != nil {
		redirectDetails = &RedirectDetails{
			Status:         res.Request.Response.Status,
			StatusCode:     res.Request.Response.StatusCode,
			OriginalUrl:    rawUrl,
			DestinationUrl: res.Request.URL.String(),
		}
	}

	parseTime := time.Now()

	return SiteResponse{
		Url:             rawUrl,
		UrlQuery:        urlQuery,
		StatusCode:      res.StatusCode,
		RedirectDetails: redirectDetails,
		Status:          res.Status,
		FetchTime:       fetchTimeStr,
		ContentLength:   contentLength,
		ContentType:     contentType,
		HtmlDetails:     parseHtml(res),
		ParseTime:       time.Since(parseTime).String(),
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

		case "form":
			{
				// collect all the relevant tags within this form
				for c := range n.Descendants() {
					form := FormTag{}

					if slices.Contains(validFormChildren, c.Data) {
						if htmlData.Forms == nil {
							htmlData.Forms = &[]FormTag{}
						}

						if c.Data == "input" || c.Data == "textarea" || c.Data == "button" {
							form.Fields = FormChild{
								Field: c.Data,
							}

							// look for type, name, required, placeholder, value
							for _, attr := range c.Attr {
								if attr.Key == "" || attr.Val == "" {
									continue
								}

								switch attr.Key {
								case "type":
									form.Fields.Type = &attr.Val
								case "name":
									form.Fields.Name = &attr.Val
								case "required":
									b, err := strconv.ParseBool(attr.Val)
									if err != nil {
										continue
									}
									form.Fields.Required = &b
								case "placeholder":
									form.Fields.Placeholder = &attr.Val
								case "value":
									form.Fields.Value = &attr.Val
								case "id":
									form.Fields.Id = &attr.Val
								}
							}

							// add text node inside button to button field value
							if c.Data == "button" && c.FirstChild != nil && c.FirstChild.Data != "" {
								t := extractWords(c)
								s := strings.Join(t, " ")
								form.Fields.Value = &s
							}

							*htmlData.Forms = append(*htmlData.Forms, form)
						}

						if c.Data == "label" {
							form.Fields = FormChild{
								Field: "label",
							}
							for _, attr := range c.Attr {
								if attr.Key == "" || attr.Val == "" {
									continue
								}

								if attr.Key == "for" {
									form.Fields.For = &attr.Val
								}
							}
							*htmlData.Forms = append(*htmlData.Forms, form)

						}
					}
				}
			}

		case "title":
			{
				words := extractWords(n)
				if len(words) == 0 {
					continue
				}
				text := strings.Join(words, " ")

				if htmlData.Title == nil || *htmlData.Title == "" {
					htmlData.Title = &text
				}
			}

		case "meta":
			{
				if len(n.Attr) == 0 {
					continue
				}

				b := false
				for _, v := range n.Attr {

					// DESCRIPTION
					if v.Key == "name" && v.Val == "description" && (htmlData.Description == nil || *htmlData.Description == "") {
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

					// OPENGRAPH
					if slices.Contains(validOpenGraphProperties, v.Val) {
						for _, v2 := range n.Attr {
							if v2.Key == "content" {
								// create the map if nil
								if htmlData.OpenGraphTags == nil {
									o := make(OpenGraphTag)
									htmlData.OpenGraphTags = &o
								}

								(*htmlData.OpenGraphTags)[v.Val] = strings.TrimSpace(v2.Val)
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
				words := extractWords(n)
				if len(words) == 0 {
					continue
				}
				text := strings.Join(words, " ")

				if htmlData.SpanTags == nil {
					htmlData.SpanTags = &[]string{text}
				} else {
					*htmlData.SpanTags = append(*htmlData.SpanTags, text)
				}
			}

		case "p":
			{
				words := extractWords(n)
				if len(words) == 0 {
					continue
				}
				text := strings.Join(words, " ")

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
				words := extractWords(n)
				if len(words) == 0 {
					continue
				}
				text := strings.Join(words, " ")

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
						break
					}
				}

				if href == "" {
					continue
				}

				if htmlData.LinkTags == nil {
					htmlData.LinkTags = &[]string{href}
				} else {
					*htmlData.LinkTags = append(*htmlData.LinkTags, href)
				}
			}

		case "ul", "ol":
			{
				// loop through all the lis of the list
				for li := range n.Descendants() {
					if li.Data == "li" {
						words := extractWords(n)
						if len(words) == 0 {
							continue
						}
						text := strings.Join(words, " ")

						switch n.Data {
						case "ol":
							{
								if htmlData.Lists == nil || htmlData.Lists.OrderedLists == nil {
									htmlData.Lists = &ListTag{OrderedLists: &[]string{text}}
								} else {
									x := htmlData.Lists
									*x.OrderedLists = append(*x.OrderedLists, text)
									htmlData.Lists = x
								}
							}

						case "ul":
							{
								if htmlData.Lists == nil || htmlData.Lists.UnorderedLists == nil {
									htmlData.Lists = &ListTag{UnorderedLists: &[]string{text}}
								} else {
									x := htmlData.Lists
									*x.UnorderedLists = append(*x.UnorderedLists, text)
									htmlData.Lists = x
								}
							}
						}
					}

				}
			}

		case "table":
			{
				headers := []string{}
				rows := []string{}

				// extract all text from the <thead> and <tbody> tags
				for tabletags := range n.Descendants() {
					if tabletags.Data == "thead" {
						headersText := extractWords(tabletags)
						if len(headersText) > 0 {
							headers = append(headers, headersText...)
						}
					}

					if tabletags.Data == "tbody" {
						bodyText := extractWords(tabletags)
						if len(bodyText) > 0 {
							rows = append(rows, bodyText...)
						}
					}
				}

				if len(headers) > 0 {
					if htmlData.Tables == nil {
						htmlData.Tables = &[]TableTag{{Headers: &headers}}
					} else {
						*htmlData.Tables = append(*htmlData.Tables, TableTag{Headers: &headers})
					}
				}

				if len(rows) > 0 {
					if htmlData.Tables == nil {
						htmlData.Tables = &[]TableTag{{Rows: &rows}}
					} else {
						*htmlData.Tables = append(*htmlData.Tables, TableTag{Rows: &rows})
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
	// the "device" url query for the site url query wont be picked up with golang net/http for some reason

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
	// its used for getting things like img src and script src urls...

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
		src = fmt.Sprintf("%v://%v%v", response.Request.URL.Scheme, response.Request.URL.Host, str)
	} else {
		// points to external site
		src = fmt.Sprintf("%v://%v%v", url.Scheme, url.Host, url.Path)
	}

	return src
}

func extractWords(n *html.Node) []string {
	// walk down the dom tree until you find the inner-most child
	// with a type of "text" node

	// more sibling to sibling across nodes and collect all text

	// ex: the h1 tag below should extract "Business openings and closings in August 2026 roundup"
	// <h1>
	//   Business <span>openings and <a href="#">closings</a></span> in
	//   <em>August</em>
	//   <span><b>2026</b> roundup</span>
	// </h1>

	words := []string{}

	for node := range n.Descendants() {
		data := strings.TrimSpace(node.Data)

		if data == "" {
			continue
		}

		if node.Type == html.TextNode {
			words = append(words, data)
		}
	}

	return words
}
