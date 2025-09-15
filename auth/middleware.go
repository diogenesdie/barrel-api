package auth

import (
	"barrel-api/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func AuthecationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		if token == "" {
			writeError(w, "Missing authentication token", http.StatusBadRequest, 1001)
			return
		}

		tokenParts := strings.Split(token, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			writeError(w, "Invalid authentication token format", http.StatusBadRequest, 1002)
			return
		}

		token = tokenParts[1]

		claims, err := VerifyToken(token)
		if err != nil {
			writeError(w, "Invalid or expired authentication token", http.StatusUnauthorized, 1003)
			return
		}

		userID := claims["user_id"].(float64)
		r.Header.Set("user_id", fmt.Sprintf("%d", int(userID)))

		next.ServeHTTP(w, r)
	})
}

func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, message string, status, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := model.Response{
		Message: message,
		Code:    code,
		Status:  status,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
