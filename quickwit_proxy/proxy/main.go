package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const QUICKWIT_BASE_URL = "http://localhost:7280"

func extractEmailFromJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid jwt format")
	}

	payloadPart := parts[1]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return "", err
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", err
	}

	email, ok := claims["email"].(string)
	if !ok {
		return "", fmt.Errorf("email claim not found")
	}

	return email, nil
}

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

	query, ok := queryObj["query"].(map[string]any)
	if !ok {
		return rawBody
	}

	boolQuery, ok := query["bool"].(map[string]any)
	if !ok {
		return rawBody
	}

	filters, ok := boolQuery["filter"].([]any)
	if !ok {
		return rawBody
	}

	for _, f := range filters {
		filterMap, ok := f.(map[string]any)
		if !ok {
			continue
		}

		if qs, exists := filterMap["query_string"]; exists {
			qsMap, ok := qs.(map[string]any)
			if !ok {
				continue
			}

			queryStr, ok := qsMap["query"].(string)
			if !ok {
				continue
			}

			qsMap["query"] = "(" + queryStr + ") AND (container_id:x)"
		}
	}

	modifiedQuery, err := json.Marshal(queryObj)
	if err != nil {
		return rawBody
	}

	return bytes.Join([][]byte{
		meta,
		modifiedQuery,
		[]byte(""),
	}, []byte("\n"))
}

func main() {
	target, err := url.Parse(QUICKWIT_BASE_URL)
	if err != nil {
		log.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Set backend target
			pr.SetURL(target)

			// Only rewrite POST requests
			if pr.In.Method != http.MethodPost {
				return
			}

			// Read original request body
			bodyBytes, err := io.ReadAll(pr.In.Body)
			if err != nil {
				return
			}
			pr.In.Body.Close()

			// Rewrite body
			updatedBody := rewriteMsearchBody(bodyBytes)

			// Replace outgoing body
			pr.Out.Body = io.NopCloser(bytes.NewReader(updatedBody))
			pr.Out.ContentLength = int64(len(updatedBody))
			pr.Out.Header.Set("Content-Length", strconv.Itoa(len(updatedBody)))
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		jwtToken := r.Header.Get("X-Grafana-Id")
		if jwtToken == "" {
			http.Error(w, "Missing X-Grafana-Id", http.StatusUnauthorized)
			return
		}

		email, err := extractEmailFromJWT(jwtToken)
		if err != nil {
			http.Error(w, "Invalid JWT", http.StatusUnauthorized)
			return
		}

		fmt.Println("Request from user email:", email)

		// Continue normal processing
		proxy.ServeHTTP(w, r)
	})

	log.Println("proxy running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
