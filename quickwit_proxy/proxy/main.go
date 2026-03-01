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
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/bcrypt"
)

var (
	QUICKWIT_BASE_URL = "http://localhost:7280"
	FRAPPE_BASE_URL   = fmt.Sprintf("https://%s", os.Getenv("FRAPPE_HOST"))

	BIND_PORT = getEnv("PROXY_BIND_PORT", "8080")
	DOMAIN    = os.Getenv("PROXY_DOMAIN")
)

var (
	logUserCache = make(map[string]map[string]any)
	cacheMutex   sync.RWMutex
	jwksMap      = make(map[string]*keyfunc.JWKS)
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

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func validateBasicAuth(r *http.Request) bool {

	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return false
	}

	parts := strings.SplitN(string(payload), ":", 2)
	if len(parts) != 2 {
		return false
	}

	username := parts[0]
	password := parts[1]

	expectedUser := os.Getenv("QUICKWIT_ADMIN_USERNAME")
	expectedHash := os.Getenv("QUICKWIT_ADMIN_HASHED_PASSWORD")

	if username != expectedUser {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(expectedHash), []byte(password))
	return err == nil
}

func initJWKS() {
	raw := os.Getenv("JWKS_MAP")
	if raw == "" {
		log.Fatal("JWKS_MAP not set")
	}

	var issuerMap map[string]string
	if err := json.Unmarshal([]byte(raw), &issuerMap); err != nil {
		log.Fatalf("Invalid JWKS_MAP JSON: %v", err)
	}

	for issuer, jwksURL := range issuerMap {
		jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
			RefreshInterval: time.Hour,
			RefreshTimeout:  10 * time.Second,
		})
		if err != nil {
			log.Fatalf("Failed to load JWKS for issuer %s: %v", issuer, err)
		}
		jwksMap[issuer] = jwks
		log.Printf("Loaded JWKS for issuer: %s", issuer)
	}
}

func verifyJWT(tokenString string) (jwt.MapClaims, error) {
	parser := jwt.Parser{}
	unverifiedToken, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token")
	}

	claims := unverifiedToken.Claims.(jwt.MapClaims)
	iss, ok := claims["iss"].(string)
	if !ok {
		return nil, fmt.Errorf("issuer missing in token")
	}

	jwks, exists := jwksMap[iss]
	if !exists {
		return nil, fmt.Errorf("unknown issuer: %s", iss)
	}

	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token.Claims.(jwt.MapClaims), nil
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

	bodyBytes, _ := json.Marshal(requestBody)

	url := fmt.Sprintf("%s/api/method/generic_logger.api.get_log_user_meta", FRAPPE_BASE_URL)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
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

	cacheMutex.Lock()
	logUserCache[email] = logUserData
	cacheMutex.Unlock()

	return logUserData, nil
}

func fixRangeField(filter map[string]any) {
	if r, ok := filter["range"].(map[string]any); ok {
		if _, exists := r[""]; exists {
			r["time"] = r[""]
			delete(r, "")
		}
	}
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

	var filters []any

	switch f := boolQuery["filter"].(type) {
	case []any:
		filters = f
	case map[string]any:
		filters = []any{f}
	default:
		filters = []any{}
	}

	for _, item := range filters {
		if m, ok := item.(map[string]any); ok {
			fixRangeField(m)
		}
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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if validateBasicAuth(r) {
			proxy.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("X-Grafana-Id")
		if token == "" {
			log.Println("Missing token")

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","authenticated":false}`))
			return
		}

		claims, err := verifyJWT(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		email, ok := claims["email"].(string)
		if !ok {
			http.Error(w, "Email missing", http.StatusUnauthorized)
			return
		}

		blockedPaths := []string{"/ui"}
		for _, path := range blockedPaths {
			if strings.Contains(r.URL.Path, path) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		logUserData, err := fetchLogUser(token, email)
		if err != nil {
			http.Error(w, "Frappe lookup failed", http.StatusInternalServerError)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "logUserData", logUserData))
		proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    ":" + BIND_PORT,
		Handler: handler,
	}

	log.Printf("Proxy running on port %s", BIND_PORT)

	if DOMAIN != "" {
		log.Printf("TLS enabled for domain %s", DOMAIN)

		m := &autocert.Manager{
			Cache:      autocert.DirCache("certs"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(DOMAIN),
		}

		server.TLSConfig = m.TLSConfig()

		go http.ListenAndServe(":80", m.HTTPHandler(nil))
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
