package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"test/auth"

	"github.com/casbin/casbin/v2"
)

// middleware/middleware.go
var (
	publicPaths = map[string]map[string]bool{
		"/v1/api/register": {
			"POST": true, // ListCars
		},
		"/v1/api/login": {
			"POST": true, // GetCarById
		},
	}
)

func isPublicPath(path, method string) bool {
	for publicPath, methods := range publicPaths {
		if strings.HasPrefix(path, strings.Split(publicPath, "{")[0]) {
			if methods[method] {
				return true
			}
		}
	}
	return strings.HasPrefix(path, "/swagger-ui/")
}
func AuthMiddleware(enforcer *casbin.Enforcer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the path and method are public
			if isPublicPath(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			fmt.Println(r.URL.Path, r.Method)

			// Get token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			_, role, err := auth.GetUserIdFromToken(authHeader)
			if err != nil {
				http.Error(w, "Authorization error", http.StatusInternalServerError)
				fmt.Println(err)
				return
			}

			// Check permission
			allowed, err := enforcer.Enforce(role, r.URL.Path, r.Method)
			fmt.Println(role, r.URL.Path, r.Method)
			if err != nil {
				http.Error(w, "Authorization error", http.StatusInternalServerError)
				fmt.Println(err)
				return
			}
			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
