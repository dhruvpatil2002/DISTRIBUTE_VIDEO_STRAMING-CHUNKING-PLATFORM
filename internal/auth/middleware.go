
package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
)

func Middleware(userService *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				log.Println("AUTH: Authorization header missing")
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Fields(authHeader)

			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				log.Println("AUTH: Invalid Bearer format")
				http.Error(w, "Invalid Bearer format", http.StatusUnauthorized)
				return
			}

			user, err := userService.GetUserFromToken(
				r.Context(),
				parts[1],
			)

			if err != nil {
				log.Printf("AUTH: JWT validation failed: %v", err)
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := SetUserID(r.Context(), user.ID)
			ctx = SetUserRole(ctx, user.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}


