package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	QUICKWIT_BASE_URL = "http://localhost:7280"
	JWKS_URL          = "http://188.245.72.65:3000/api/signing-keys/keys"
	FRAPPE_BASE_URL   = "https://logger.kwscloud.in"
)

var frappeMetaFields = map[string]bool{
	"name":        true,
	"owner":       true,
	"creation":    true,
	"modified":    true,
	"modified_by": true,
	"docstatus":   true,
	"idx":         true,
	"doctype":     true,
	"user":        true,
}

type FrappeResponse struct {
	Data map[string]any `json:"data"`
}

var jwks *keyfunc.JWKS

func initJWKS() {
	var err error

	jwks, err = keyfunc.Get(JWKS_URL, keyfunc.Options{
		RefreshInterval: time.Hour, // auto refresh
		RefreshTimeout:  10 * time.Second,
	})

	if err != nil {
		log.Fatalf("Failed to create JWKS: %v", err)
	}

	log.Println("JWKS initialized")
}

func verifyJWT(tokenString string) (jwt.MapClaims, error) {

	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

func fetchLogUser(email string) (map[string]any, error) {

	apiToken := os.Getenv("API_TOKEN")
	apiSecret := os.Getenv("API_SECRET")

	if apiToken == "" || apiSecret == "" {
		return nil, fmt.Errorf("missing frappe credentials")
	}

	url := fmt.Sprintf("%s/api/resource/Log%%20User/%s", FRAPPE_BASE_URL, email)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", apiToken, apiSecret))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frappe returned %d", resp.StatusCode)
	}

	var result FrappeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func rewriteMsearchBody(rawBody []byte, frappeData map[string]any) []byte {

	lines := bytes.Split(rawBody, []byte("\n"))
	if len(lines) < 2 {
		return rawBody
	}

	meta := lines[0]

	var queryObj map[string]any
	if err := json.Unmarshal(lines[1], &queryObj); err != nil {
		return rawBody
	}

	query := queryObj["query"].(map[string]any)
	boolQuery := query["bool"].(map[string]any)

	filters, _ := boolQuery["filter"].([]any)

	for key, value := range frappeData {
		if frappeMetaFields[key] || value == nil {
			continue
		}

		filters = append(filters, map[string]any{
			"term": map[string]any{
				key: value,
			},
		})
	}

	boolQuery["filter"] = filters

	modifiedQuery, _ := json.Marshal(queryObj)

	return bytes.Join([][]byte{meta, modifiedQuery, []byte("")}, []byte("\n"))
}

func main() {

	initJWKS()

	target, _ := url.Parse(QUICKWIT_BASE_URL)

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {

			pr.SetURL(target)

			if pr.In.Method != http.MethodPost {
				return
			}

			bodyBytes, _ := io.ReadAll(pr.In.Body)
			pr.In.Body.Close()

			logUserData, _ := pr.In.Context().Value("logUserData").(map[string]any)

			updatedBody := rewriteMsearchBody(bodyBytes, logUserData)

			pr.Out.Body = io.NopCloser(bytes.NewReader(updatedBody))
			pr.Out.ContentLength = int64(len(updatedBody))
			pr.Out.Header.Set("Content-Length", strconv.Itoa(len(updatedBody)))
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("X-Grafana-Id")
		if token == "" {
			log.Println("Missing token")
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		claims, err := verifyJWT(token)
		if err != nil {
			log.Println("Invalid token", err.Error())
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		email, ok := claims["email"].(string)
		if !ok {
			log.Println("Email missing")
			http.Error(w, "Email missing", http.StatusUnauthorized)
			return
		}

		logUserData, err := fetchLogUser(email)
		if err != nil {
			http.Error(w, "Frappe lookup failed", http.StatusInternalServerError)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "logUserData", logUserData))

		proxy.ServeHTTP(w, r)
	})

	log.Println("proxy running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
