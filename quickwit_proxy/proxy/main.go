package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const QUICKWIT_BASE_URL = "http://localhost:7280"

func rewriteMsearchBody(rawBody []byte) []byte {
	lines := bytes.Split(rawBody, []byte("\n"))

	if len(lines) < 2 || len(bytes.TrimSpace(lines[1])) == 0 {
		return rawBody
	}

	meta := lines[0]

	var queryObj map[string]any
	if err := json.Unmarshal(lines[1], &queryObj); err != nil {
		return rawBody
	}

	query := queryObj["query"].(map[string]any)
	boolQuery := query["bool"].(map[string]any)
	filters := boolQuery["filter"].([]any)

	for _, f := range filters {
		filterMap := f.(map[string]any)
		if qs, exists := filterMap["query_string"]; exists {
			qsMap := qs.(map[string]any)
			queryStr := qsMap["query"].(string)
			qsMap["query"] = "(" + queryStr + ") AND (container_id:x)"
		}
	}

	modifiedQuery, _ := json.Marshal(queryObj)

	return bytes.Join([][]byte{
		meta,
		modifiedQuery,
		[]byte(""),
	}, []byte("\n"))
}

func main() {
	target, _ := url.Parse(QUICKWIT_BASE_URL)

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize transport (timeout)
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 5 * time.Second,
	}

	proxy.ModifyResponse = nil

	originalDirector := proxy.Rewrite

	proxy.Rewrite = func(req *httputil.ProxyRequest) {
		originalDirector(req)

		// Read body
		bodyBytes, err := io.ReadAll(req.Out.Body)
		if err != nil {
			return
		}

		req.Out.Body.Close()

		updatedBody := rewriteMsearchBody(bodyBytes)

		req.Out.Body = io.NopCloser(bytes.NewReader(updatedBody))
		req.Out.ContentLength = int64(len(updatedBody))
	}

	log.Println("Enterprise Proxy running on :8080")
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
