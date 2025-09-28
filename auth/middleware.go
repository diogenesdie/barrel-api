package auth

import (
	"barrel-api/model"
	"barrel-api/repository"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

const (
	sessionKey contextKey = "session"
	userIDKey  contextKey = "user_id"
)

func AuthenticationMiddleware(sessionRepo *repository.SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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

			session, err := sessionRepo.ValidateSession(token)
			if err != nil {
				var message string
				var code int

				switch err {
				case repository.ErrSessionNotFound:
					message = "Session not found"
					code = 1004
				case repository.ErrSessionExpired:
					message = "Session expired"
					code = 1005
				case repository.ErrSessionInactive:
					message = "Session inactive"
					code = 1006
				default:
					message = "Session validation failed"
					code = 1007
				}

				writeError(w, message, http.StatusUnauthorized, code)
				return
			}
			ctx := context.WithValue(r.Context(), sessionKey, session)
			ctx = context.WithValue(ctx, userIDKey, session.UserID)

			userID := claims["user_id"].(float64)
			r.Header.Set("user_id", fmt.Sprintf("%d", int(userID)))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

func GetSessionFromContext(r *http.Request) (*model.Session, bool) {
	session, ok := r.Context().Value(sessionKey).(*model.Session)
	return session, ok
}

func GetUserIDFromContext(r *http.Request) (uint64, bool) {
	userID, ok := r.Context().Value(userIDKey).(uint64)
	return userID, ok
}
