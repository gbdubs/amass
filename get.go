package amass

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func (req *GetRequest) Get() (*GetResponse, error) {
	resp, isMissing, err := req.readMemo()
	if err != nil {
		return resp, fmt.Errorf("Memo-check for %s %s returned: %v", req.Site, req.RequestKey, err)
	}
	if !isMissing {
		return resp, nil
	}
	resp = &GetResponse{}
	hq, err := http.NewRequest("GET", req.URL, nil)
	if err != nil {
		return resp, fmt.Errorf("Constructing http request to %s: %v", req.URL, err)
	}
	hr, err := http.DefaultClient.Do(hq)
	if err != nil {
		return resp, fmt.Errorf("Actual http request failure to %s: %v", req.URL, err)
	}
	defer hr.Body.Close()

	resp.StatusCode = hr.StatusCode
	resp.Status = hr.Status

	if resp.shouldReturnError() {
		return resp, fmt.Errorf("Request to %s failed with status %d %s.", req.URL, hr.StatusCode, hr.Status)
	}

	asBytes, err := ioutil.ReadAll(hr.Body)
	if err != nil {
		return resp, fmt.Errorf("Reading request body failed for %s: %v", req.URL, err)
	}
	responseBody := string(asBytes)

	if strings.Contains(responseBody, "<!doctype html>") || strings.Contains(responseBody, "<!DOCTYPE html>") {
		d, err := goquery.NewDocumentFromReader(strings.NewReader(responseBody))
		if err != nil {
			return resp, fmt.Errorf("Document parse of %s failed: %v", req.URL, err)
		}

		d.Find("script").ReplaceWithHtml("<!-- Removed script -->")
		d.Find("style").ReplaceWithHtml("<!-- Removed style -->")
		d.Find("link").ReplaceWithHtml("<!-- Removed link -->")
		d.Find("img").ReplaceWithHtml("<!-- Removed img -->")
		d.Find("svg").ReplaceWithHtml("<!-- Removed svg -->")
		d.Find("image:image").ReplaceWithHtml("<!-- Removed image:image -->")
		responseBody, err = d.Html()
		if err != nil {
			return resp, fmt.Errorf("Converting to html failed for document at %s: %v", req.URL, err)
		}
	}

	resp.Site = req.Site
	resp.RequestKey = req.RequestKey
	resp.Version = version
	resp.URL = req.URL
	resp.ResponseBody = responseBody
	resp.Attribution = req.Attribution
	resp.Attribution.OriginUrl = req.URL
	resp.Attribution.CollectedAt = time.Now()
	resp.Attribution.OriginalTitle = resp.AsDocument().Find("title").First().Text()
	resp.RoundTripData = req.RoundTripData

	if resp.shouldMemo() {
		err = resp.writeMemo()
		if err != nil {
			return resp, fmt.Errorf("Memoization write failed for %s: %v", resp.URL, err)
		}
	}

	return resp, nil
}

func (r *GetResponse) shouldReturnError() bool {
	sc := r.StatusCode
	if 200 <= sc && sc < 400 {
		return false
	}
	if sc == 404 /* Not Found */ || sc == 410 /* Gone */ {
		return false
	}
	if 400 <= sc && sc < 600 {
		return true
	}
	panic(fmt.Errorf("Unexpected HTTP Response Code %d at %s", sc, r.URL))
}

func (r *GetResponse) shouldMemo() bool {
	sc := r.StatusCode
	if 200 <= sc && sc < 400 {
		return true
	}
	if sc == 404 /* Not found */ || sc == 410 /* Gone */ {
		return true
	}
	if 400 <= sc && sc < 600 {
		return false
	}
	panic(fmt.Errorf("Unexpected HTTP Response Code %d at %s", sc, r.URL))
}
