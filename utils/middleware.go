package utils

import (
	"hm-dianping-go/dao"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	LoginTokenBlackList = "dianping:user:blacklist:"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		//获取Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			ErrorResponse(c, http.StatusUnauthorized, "未提供认证token")
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ErrorResponse(c, http.StatusUnauthorized, "认证格式错误，请使用 Bearer token")
			c.Abort()
			return
		}
		token := parts[1]
		ctx := c.Request.Context()
		key := LoginTokenBlackList + token
		exists, err := dao.Redis.Exists(ctx, key).Result()
		if err == nil && exists == 1 {
			ErrorResponse(c, http.StatusUnauthorized, "token已失效，请重新登录")
			c.Abort()
			return
		}
		claims, err := ParseToken(token)
		if err != nil {
			ErrorResponse(c, http.StatusUnauthorized, "无效的token")
			c.Abort()
			return
		}
		//fmt.Printf("UserID: %d\n", claims.UserID)
		c.Set("userID", claims.UserID)
		c.Set("token", token)
		c.Next()
	}
}
