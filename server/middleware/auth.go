package middleware

import (
	"send/server/service"
	"send/server/utils"

	"github.com/gin-gonic/gin"
)

func AdminAuth(adminSvc *service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Admin-Token")
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
