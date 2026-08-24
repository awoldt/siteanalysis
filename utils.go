package main

import (
	"fmt"
	"net/http"
	"time"
)

type SiteResponse struct {
	StatusCode    int
	ResponseTime  string
	ContentLength int64
	ContentType   string
}

func fetchSite(url string) (SiteResponse, error) {
	startTime := time.Now()

	res, err := http.Get(url)
	if err != nil {
		return SiteResponse{}, err
	}

	defer res.Body.Close()

	fmt.Println(res.Header)

	return SiteResponse{
		StatusCode:    res.StatusCode,
		ResponseTime:  time.Since(startTime).String(),
		ContentLength: res.ContentLength,
		ContentType:   res.Header.Get("Content-Type"),
	}, nil
}
