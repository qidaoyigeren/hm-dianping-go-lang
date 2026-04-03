package utils

import (
	"context"
	"fmt"
	"hm-dianping-go/dao"
	"net/http"
	"strings"
	"time"

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
		blacklisted, err := dao.IsTokenInBlacklist(token)
		if err == nil && blacklisted {
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

// 跨域中间件
func CORS() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
}
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		go func() {
			var userIdentifier string
			if userID, exists := c.Get("userID"); exists {
				userIdentifier = fmt.Sprintf("user:%v", userID)
			} else {
				userIdentifier = fmt.Sprintf("ip:%s", c.ClientIP())
			}
			today := time.Now().Format("2006-01-02")
			uvKey := fmt.Sprintf("uv:daily:%s", today)
			// 异步记录到Redis，避免影响请求性能
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := dao.Redis.PFAdd(ctx, uvKey, userIdentifier).Err(); err != nil {
				fmt.Printf("UV统计记录失效")
			}
			// 设置key的过期时间为7天，避免数据无限增长
			dao.Redis.Expire(ctx, uvKey, 7*24*time.Hour)
		}()
	}
}
