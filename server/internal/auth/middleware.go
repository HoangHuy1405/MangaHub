package auth

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	"mangahub/pkg/utils"
)

func JWTMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("[JWT] Missing Authorization header")
			utils.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Println("[JWT] Invalid Authorization header format")
			utils.Unauthorized(c, "Authorization header must be in format: Bearer <token>")
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			log.Printf("[JWT] Invalid token: %v", err)
			utils.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Println("[JWT] Failed to parse token claims")
			utils.Unauthorized(c, "Invalid token claims")
			c.Abort()
			return
		}

		userID := int(claims["user_id"].(float64))
		username := claims["username"].(string)
		c.Set("user_id", userID)
		c.Set("username", username)

		log.Printf("[JWT] Authenticated user: id=%d, username=%s", userID, username)
		c.Next()
	}
}
