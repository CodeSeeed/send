package middleware

import (
	"send/server/service"
	"send/server/utils"

	"github.com/gin-gonic/gin"
)

func AdminAuth(adminSvc *service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// The admin session lives exclusively in the HttpOnly cookie. The legacy
		// X-Admin-Token header is intentionally not accepted: a token that
		// reaches JavaScript (response body, storage) must not be usable.
		token, err := c.Cookie("admin_token")
		if err != nil || token == "" {
			utils.Error(c, 401, "未登录")
			c.Abort()
			return
		}
		adminID, err := adminSvc.Auth(token)
		if err != nil {
			utils.Error(c, 401, err.Error())
			c.Abort()
			return
		}
		c.Set("admin_id", adminID)
		c.Next()
	}
}
