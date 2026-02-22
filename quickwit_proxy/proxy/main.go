package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const QUICKWIT_BASE_URL = "http://localhost:7280"

type FrappeResponse struct {
	Data map[string]any `json:"data"`
}

func fetchLogUser(email string) (map[string]any, error) {
	apiToken := os.Getenv("API_TOKEN")
	apiSecret := os.Getenv("API_SECRET")

	if apiToken == "" || apiSecret == "" {
		return nil, fmt.Errorf("API_TOKEN or API_SECRET not set")
	}

	frappeURL := fmt.Sprintf(
		"http://188.245.72.102:8000/api/resource/Log%%20User/%s",
		email,
	)

	req, err := http.NewRequest("GET", frappeURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", apiToken, apiSecret))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frappe returned status %d", resp.StatusCode)
	}

	var frappeResp FrappeResponse
	if err := json.NewDecoder(resp.Body).Decode(&frappeResp); err != nil {
		return nil, err
	}

	return frappeResp.Data, nil
}

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

func rewriteMsearchBody(rawBody []byte, frappeData map[string]any) []byte {
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
		filters = []any{}
	}

	// Inject all fields from Log User doc
	for key, value := range frappeData {

		if key == "doctype" || key == "name" {
			continue
		}

		filter := map[string]any{
			"term": map[string]any{
				key: value,
			},
		}

		filters = append(filters, filter)
	}

	boolQuery["filter"] = filters

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

			pr.SetURL(target)

			if pr.In.Method != http.MethodPost {
				return
			}

			bodyBytes, err := io.ReadAll(pr.In.Body)
			if err != nil {
				return
			}
			pr.In.Body.Close()

			logUserData, _ := pr.In.Context().Value("logUserData").(map[string]any)

			updatedBody := rewriteMsearchBody(bodyBytes, logUserData)

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

		log.Println("Request from user email:", email)

		// Fetch Log User document
		logUserData, err := fetchLogUser(email)
		if err != nil {
			http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
			return
		}

		// Inject into context so Rewrite can use it
		r = r.WithContext(context.WithValue(r.Context(), "logUserData", logUserData))

		proxy.ServeHTTP(w, r)
	})

	log.Println("proxy running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
