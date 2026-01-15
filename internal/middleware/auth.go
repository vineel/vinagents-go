package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vineel/vinagents-go/pkg/jwt"
)

const (
	AuthUserKey = "auth_user"
)

// AuthUser represents the authenticated user stored in context
type AuthUser struct {
	UserID string
	Email  string
}

// Auth creates an authentication middleware
func Auth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "No token provided",
			})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		// Store user info in context
		c.Set(AuthUserKey, &AuthUser{
			UserID: claims.UserID,
			Email:  claims.Email,
		})

		c.Next()
	}
}

// GetAuthUser retrieves the authenticated user from context
func GetAuthUser(c *gin.Context) *AuthUser {
	if user, exists := c.Get(AuthUserKey); exists {
		if authUser, ok := user.(*AuthUser); ok {
			return authUser
		}
	}
	return nil
}

// MustGetAuthUser retrieves the authenticated user or panics
func MustGetAuthUser(c *gin.Context) *AuthUser {
	user := GetAuthUser(c)
	if user == nil {
		panic("auth middleware not applied or user not found")
	}
	return user
}
