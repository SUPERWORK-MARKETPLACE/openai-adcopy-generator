// Package urlcheck verifies landing/image URL accessibility before generation
// (charter §6: 접근 실패 시 생성 전에 오류 표시).
package urlcheck

import (
	"net/http"
	"net/url"
	"time"
)

type URLResult struct {
	URL              string `json:"url"`
	OK               bool   `json:"ok"`
	Status           int    `json:"status,omitempty"`
	FinalURL         string `json:"final_url,omitempty"`
	HTTPS            bool   `json:"https"`
	RedirectedDomain bool   `json:"redirected_domain"`
	Error            string `json:"error,omitempty"`
}

func Check(urls []string, timeout time.Duration) []URLResult {
	client := &http.Client{Timeout: timeout}
	out := make([]URLResult, 0, len(urls))
	for _, raw := range urls {
		out = append(out, checkOne(client, raw))
	}
	return out
}

func checkOne(client *http.Client, raw string) URLResult {
	res := URLResult{URL: raw}
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "adcopy-urlcheck/1.0")
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	final := resp.Request.URL
	res.Status = resp.StatusCode
	res.FinalURL = final.String()
	res.HTTPS = final.Scheme == "https"
	if orig, err := url.Parse(raw); err == nil {
		res.RedirectedDomain = orig.Hostname() != final.Hostname()
	}
	res.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
	return res
}
