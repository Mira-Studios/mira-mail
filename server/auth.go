package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const usernameContextKey contextKey = "username"

// APIKeyMiddleware validates API key on requests
func APIKeyMiddleware(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Expect: Bearer <api_key>
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(apiKey)) != 1 {
			http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
			return
		}

		// Extract and verify user token
		username := r.Header.Get("X-Username")
		userToken := r.Header.Get("X-User-Token")
		
		if username != "" && userToken != "" {
			if !VerifyUserToken(apiKey, username, userToken) {
				http.Error(w, `{"error":"invalid user token"}`, http.StatusUnauthorized)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), usernameContextKey, username))
		}

		next(w, r)
	}
}

// GetUsernameFromContext retrieves the authenticated username from request context
func GetUsernameFromContext(r *http.Request) string {
	if username, ok := r.Context().Value(usernameContextKey).(string); ok {
		return username
	}
	return ""
}

// GenerateUserToken creates a user-specific token using HMAC
// This binds the username to the API key, preventing tampering
func GenerateUserToken(apiKey, username string) string {
	h := hmac.New(sha256.New, []byte(apiKey))
	h.Write([]byte(username))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyUserToken checks if the provided token matches the expected token for the username
func VerifyUserToken(apiKey, username, token string) bool {
	expected := GenerateUserToken(apiKey, username)
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

// CORS middleware
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Username, X-User-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
