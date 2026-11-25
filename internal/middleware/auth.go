package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
	UserRoleKey    contextKey = "role"
)

func AuthMiddleware(jwtSecret []byte, requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success": false, "message": "Token tidak valid atau sudah kedaluwarsa", "error": {"code": "UNAUTHORIZED", "details": []}}`))
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success": false, "message": "Token tidak valid atau sudah kedaluwarsa", "error": {"code": "UNAUTHORIZED", "details": []}}`))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success": false, "message": "Token tidak valid", "error": {"code": "UNAUTHORIZED", "details": []}}`))
				return
			}

			role, _ := claims["role"].(string)
			if requiredRole != "" && role != requiredRole {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"success": false, "message": "Anda tidak memiliki akses ke resource ini", "error": {"code": "FORBIDDEN", "details": []}}`))
				return
			}

			userID, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), UserContextKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
