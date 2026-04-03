package router

import (
	"hm-dianping-go/handler"
	"hm-dianping-go/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	//添加中间件
	r.Use(utils.CORS())
	r.Use(utils.Logger())
	r.Use(utils.Recovery())
	// API路由组
	api := r.Group("/api")
	{
		// 用户相关路由
		userGroup := api.Group("/user")
		{
			userGroup.POST("/code", handler.SendCode)
			userGroup.POST("/register", handler.UserRegister)
			userGroup.POST("/login", handler.UserLogin)
			userGroup.POST("/logout", utils.Auth(), handler.UserLogout)
			userGroup.GET("/me", utils.Auth(), handler.GetUserInfo)
			userGroup.PUT("/update", utils.Auth(), handler.UpdateUserInfo)
		}
		// 商铺相关路由
		shopGroup := api.Group("/shop")
		{
			shopGroup.GET("/list", handler.GetShopList)
			shopGroup.GET("/:id", handler.GetShopById)
			shopGroup.GET("/of/type", handler.GetShopByType)
			shopGroup.GET("/of/name", handler.GetShopByName)
			shopGroup.POST("", handler.SaveShop)
			shopGroup.PUT("", handler.UpdateShop)
			shopGroup.GET("/:id/nearby", utils.Auth(), handler.GetNearbyShops) // 获取某个商铺附近的商铺
		}
		// 商铺类型相关路由
		shopTypeGroup := api.Group("/shop-type")
		{
			shopTypeGroup.GET("/list", handler.GetShopTypeList)
		}
		// 优惠券相关路由
		voucherGroup := api.Group("/voucher")
		{
			voucherGroup.GET("/list/:shopId", handler.GetVoucherList)
			voucherGroup.POST("", handler.AddVoucher)
			voucherGroup.POST("/seckill", handler.AddSeckillVoucher)
			voucherGroup.GET("/seckill/:id", handler.GetSeckillVoucher)
		}
		// 优惠券订单相关路由
		voucherOrderGroup := api.Group("/voucher-order")
		{
			voucherOrderGroup.POST("/seckill/:id", utils.Auth(), handler.SeckillVoucher)
		}
		// 博客相关路由
		blogGroup := api.Group("/blog")
		{
			blogGroup.POST("", utils.Auth(), handler.CreateBlog)
			blogGroup.PUT("/like/:id", utils.Auth(), handler.LikeBlog)
			blogGroup.GET("/hot", utils.Auth(), handler.GetHotBlogList)
			blogGroup.GET("/of/me", utils.Auth(), handler.GetBlogListOfUser)
			blogGroup.GET("/:id", utils.Auth(), handler.GetBlog)
			blogGroup.GET("/of/follow", utils.Auth(), handler.GetBlogListOfFollow)
		}
		// 关注相关路由
		followGroup := api.Group("/follow")
		{
			followGroup.PUT(":id/:isFollow", utils.Auth(), handler.Follow)
			followGroup.GET("/or/not/:id", utils.Auth(), handler.IsFollow)
			followGroup.GET("/common/:id", utils.Auth(), handler.GetCommonFollows)
		}
		// 统计相关路由
		statGroup := api.Group("/stat")
		{
			statGroup.GET("/uv/today", handler.GetTodayUV)
			statGroup.GET("/uv/daily", handler.GetDailyUV)
			statGroup.GET("/uv/recent", handler.GetRecentUV)
			statGroup.GET("/uv/range", handler.GetUVRange)
		}
	}
	// 健康检查
	r.GET("/health", handler.HealthCheck)
	return r
}
