package middlewares

import (
	"errors"
	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/model"
	"pkuphysu-backend/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth() func(c *gin.Context) {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			utils.RespondError(c, 401, "Unauthorized", errors.New("unauthorized"))
			c.Abort()
			return
		}
		userClaims, err := utils.ParseToken(token)
		if err != nil {
			utils.RespondError(c, 401, "invalid token", err)
			c.Abort()
			return
		}
		user, err := db.GetUserById(userClaims.UserID)
		if err != nil {
			utils.RespondError(c, 401, "invalid token", err)
			c.Abort()
			return
		}
		if user.Disabled || user.PwdTS != userClaims.PwdTS {
			utils.RespondError(c, 401, "invalid token", errors.New("token has been revoked"))
			c.Abort()
			return
		}
		c.Set("CurrentUser", user)

		c.Next()
	}
}

// OptionalAuth exposes the current user when a valid token is present while
// keeping public read-only routes accessible to anonymous visitors.
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("CurrentUser", &model.User{})
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			c.Next()
			return
		}
		claims, err := utils.ParseToken(token)
		if err != nil {
			c.Next()
			return
		}
		user, err := db.GetUserById(claims.UserID)
		if err == nil && !user.Disabled && user.PwdTS == claims.PwdTS {
			c.Set("CurrentUser", user)
		}
		c.Next()
	}
}

func AuthAdmin() func(c *gin.Context) {
	return func(c *gin.Context) {
		user := c.MustGet("CurrentUser").(*model.User)

		if !user.IsAdmin() {
			utils.RespondError(c, 403, "PermissionDenied", errors.New("You are not an admin"))
			c.Abort()
			return
		} else {
			c.Next()
		}
	}
}

// AuthMember requires a registered member account or an administrator.
// The administrator is intentionally included so the role hierarchy remains
// cumulative for member-only resources such as the archive.
func AuthMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("CurrentUser").(*model.User)

		if user.Role < model.MEMBER {
			utils.RespondError(c, 403, "PermissionDenied", errors.New("member authorization required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
