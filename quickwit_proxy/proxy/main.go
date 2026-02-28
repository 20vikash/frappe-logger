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
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

var (
	QUICKWIT_BASE_URL = fmt.Sprintf("http://%s:7280", os.Getenv("QUICKWIT_HOST"))
	JWKS_URL          = fmt.Sprintf("http://%s:3000/api/signing-keys/keys", os.Getenv("GRAFANA_HOST"))
	FRAPPE_BASE_URL   = fmt.Sprintf("https://%s", os.Getenv("FRAPPE_HOST"))
)

var (
	logUserCache = make(map[string]map[string]any)
	cacheMutex   sync.RWMutex
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

type LogUserMethodResponse struct {
	Message struct {
		LogUser map[string]any `json:"log_user"`
	} `json:"message"`
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

func fixRangeField(filter map[string]any) {
	if r, ok := filter["range"].(map[string]any); ok {
		if _, exists := r[""]; exists {
			r["time"] = r[""]
			delete(r, "")
		}
	}
}

func fetchLogUser(jwtToken string, email string) (map[string]any, error) {

	cacheMutex.RLock()
	cached, exists := logUserCache[email]
	cacheMutex.RUnlock()

	if exists {
		return cached, nil
	}

	requestBody := map[string]string{
		"jwt_token": jwtToken,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s/api/method/generic_logger.api.get_log_user_meta",
		FRAPPE_BASE_URL,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frappe returned %d", resp.StatusCode)
	}

	var result LogUserMethodResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	logUserData := result.Message.LogUser
	if logUserData == nil {
		return nil, fmt.Errorf("log_user not found in response")
	}

	cacheMutex.Lock()
	logUserCache[email] = logUserData
	cacheMutex.Unlock()

	return logUserData, nil
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

	if aggs, ok := queryObj["aggs"].(map[string]any); ok {
		for _, agg := range aggs {
			if aggMap, ok := agg.(map[string]any); ok {
				if dh, ok := aggMap["date_histogram"].(map[string]any); ok {
					if dh["field"] == "" {
						dh["field"] = "time"
					}
				}
			}
		}
	}

	query := queryObj["query"].(map[string]any)
	boolQuery := query["bool"].(map[string]any)

	if filtersRaw, ok := boolQuery["filter"]; ok {

		switch f := filtersRaw.(type) {

		case map[string]any:
			fixRangeField(f)

		case []any:
			for _, item := range f {
				if m, ok := item.(map[string]any); ok {
					fixRangeField(m)
				}
			}
		}
	}

	var filters []any

	switch f := boolQuery["filter"].(type) {
	case []any:
		filters = f
	case map[string]any:
		filters = []any{f}
	default:
		filters = []any{}
	}

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

			bodyBytes, _ := io.ReadAll(pr.In.Body)
			pr.In.Body.Close()

			log.Println(string(bodyBytes))

			pr.SetURL(target)

			if pr.In.Method != http.MethodPost {
				pr.Out.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				return
			}

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

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","authenticated":false}`))
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

		logUserData, err := fetchLogUser(token, email)
		if err != nil {
			log.Println(email)
			http.Error(w, "Frappe lookup failed", http.StatusInternalServerError)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "logUserData", logUserData))

		proxy.ServeHTTP(w, r)
	})

	log.Println("proxy running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
