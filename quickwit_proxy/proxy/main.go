package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const QUICKWIT_BASE_URL = "http://localhost:7280"

var frappeMetaFields = map[string]bool{
	"name":        true,
	"owner":       true,
	"creation":    true,
	"modified":    true,
	"modified_by": true,
	"docstatus":   true,
	"idx":         true,
	"doctype":     true,
	"user":        true, // skip explicitly
}

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type FrappeResponse struct {
	Data map[string]any `json:"data"`
}

func fetchJWKS() (*JWKS, error) {
	resp, err := http.Get("http://188.245.72.65:3000/api/signing-keys/keys")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	return &jwks, nil
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

func verifyJWT(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}

	headerB64, payloadB64, signatureB64 := parts[0], parts[1], parts[2]

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return nil, err
	}

	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}

	kid, ok := header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("kid not found")
	}

	// Fetch JWKS
	jwks, err := fetchJWKS()
	if err != nil {
		return nil, err
	}

	var key *JWK
	for _, k := range jwks.Keys {
		if k.Kid == kid {
			key = &k
			break
		}
	}
	if key == nil {
		return nil, fmt.Errorf("matching key not found")
	}

	// Decode public key coordinates
	xBytes, _ := base64.RawURLEncoding.DecodeString(key.X)
	yBytes, _ := base64.RawURLEncoding.DecodeString(key.Y)

	pubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	// Hash signing input
	signingInput := headerB64 + "." + payloadB64
	hash := sha256.Sum256([]byte(signingInput))

	// Decode signature
	sigBytes, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return nil, err
	}

	if len(sigBytes) != 64 {
		return nil, fmt.Errorf("invalid signature length")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	if !ecdsa.Verify(&pubKey, hash[:], r, s) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}

	// Validate exp
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return nil, fmt.Errorf("token expired")
		}
	}

	return claims, nil
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

		// Skip metadata/system fields
		if frappeMetaFields[key] {
			continue
		}

		// Only inject non-empty values
		if value == nil {
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

		claims, err := verifyJWT(jwtToken)
		if err != nil {
			http.Error(w, "Invalid JWT", http.StatusUnauthorized)
			return
		}

		email, ok := claims["email"].(string)
		if !ok {
			http.Error(w, "Email not found in token", http.StatusUnauthorized)
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
