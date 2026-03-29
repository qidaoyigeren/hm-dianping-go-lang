package router

import (
	"hm-dianping-go/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	// TODO:添加中间件

	// API路由组
	api := r.Group("/api")
	{
		// 用户相关路由
		userGroup := api.Group("/user")
		{
			userGroup.POST("/code", handler.SendCode)
		}
		// 商铺相关路由

		// 商铺类型相关路由

		// 优惠券相关路由

		// 优惠券订单相关路由

		// 博客相关路由

		// 关注相关路由

		// 统计相关路由
	}
	return r
}
