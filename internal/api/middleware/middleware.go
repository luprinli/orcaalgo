package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/security"
)

var jwtSecret []byte
var jwtSecretOnce sync.Once

var authRepo *db.Repository
var authRepoMu sync.RWMutex

// SetAuthRepo sets the database repository used by AuthMiddleware to
// check token revocation status. Safe for concurrent use.
func SetAuthRepo(repo *db.Repository) {
	authRepoMu.Lock()
	defer authRepoMu.Unlock()
	authRepo = repo
}

func getAuthRepo() *db.Repository {
	authRepoMu.RLock()
	defer authRepoMu.RUnlock()
	return authRepo
}

func loadJWTSecret() {
	jwtSecretOnce.Do(func() {
		if jwtSecret != nil {
			return
		}
		s := os.Getenv("ORCA_JWT_SECRET")
		if s == "" {
			// HP #10: do not panic for recoverable errors. The secret is
			// validated at startup (main.go); if this path is reached it
			// degrades to an auth failure rather than a process crash.
			slog.Error("ORCA_JWT_SECRET environment variable is required but not set")
			return
		}
		jwtSecret = []byte(s)
	})
}

func GetJWTSecret() []byte {
	loadJWTSecret()
	return jwtSecret
}

func SetJWTSecret(secret []byte) { jwtSecret = secret }

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-2FA-Token")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}
		claims, err := security.ValidateToken(parts[1], GetJWTSecret())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		if repo := getAuthRepo(); repo != nil {
			if revoked, err := repo.IsTokenRevoked(context.Background(), claims.ID); err == nil && revoked {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}

type TOTPValidator func(username, code string) bool

func TwoFAMiddleware(validate TOTPValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-2FA-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "X-2FA-Token header required"})
			return
		}
		if len(token) != 6 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid 2FA token format"})
			return
		}
		username := c.GetString("username")
		if username == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Authentication required before 2FA"})
			return
		}
		if validate != nil && !validate(username, token) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid 2FA code"})
			return
		}
		c.Next()
	}
}

func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenHeader := c.GetHeader("Authorization")
		if tokenHeader == "" || len(tokenHeader) < 8 || tokenHeader[:7] != "Bearer " {
			c.Set("user_id", "")
			c.Set("username", "")
			c.Next()
			return
		}

		token := tokenHeader[7:]
		claims, err := security.ValidateToken(token, GetJWTSecret())
		if err != nil {
			c.Set("user_id", "")
			c.Set("username", "")
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}
