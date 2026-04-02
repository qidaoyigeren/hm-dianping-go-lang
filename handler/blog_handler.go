package handler

import (
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func CreateBlog(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResult("用户未登录")
		return
	}
	var req struct {
		ShopId  uint   `json:"shopId"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Images  string `json:"images"`
	}
	if err := c.ShouldBind(&req); err != nil {
		utils.ErrorResult("参数错误")
		return
	}
	result := service.CreateBlog(c.Request.Context(), userID.(uint), req.ShopId, req.Title, req.Content, req.Images)
	utils.Response(c, result)
}

func LikeBlog(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResult("用户未登录")
		return
	}
	blogID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ErrorResult("参数错误")
		return
	}
	result := service.LikeBlog(c.Request.Context(), userID.(uint), uint(blogID))
	utils.Response(c, result)
}

func GetHotBlogList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	result := service.GetHotBlogList(c.Request.Context(), userID.(uint), page, size)
	utils.Response(c, result)
}

func GetBlogListOfUser(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	result := service.GetBlogListOfUser(c.Request.Context(), userID.(uint), page, size)
	utils.Response(c, result)
}

func GetBlog(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的博客ID")
		return
	}
	result := service.GetBlogById(c.Request.Context(), uint(id), userID.(uint))
	utils.Response(c, result)
}

func GetBlogListOfFollow(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	lastID, _ := strconv.Atoi(c.DefaultQuery("lastId", "0"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "10"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	result := service.GetBlogListOfFollow(c.Request.Context(), userID.(uint), uint(lastID), offset, count)
	utils.Response(c, result)
}
