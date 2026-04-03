package handler

import (
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Follow(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	idStr := c.Param("id")
	if idStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户ID不能为空")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户ID格式不正确")
		return
	}
	var isFollow bool
	isFollow, err = strconv.ParseBool(c.Param("isFollow"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "isFollow应为bool值")
		return
	}
	result := service.Follow(c.Request.Context(), userID.(uint), uint(id), isFollow)
	utils.Response(c, result)

}

func IsFollow(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	idStr := c.Param("id")
	if idStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "查询关注ID不能为空")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "查询关注ID格式不正确")
	}
	result := service.IsFollow(c.Request.Context(), userID.(uint), uint(id))
	utils.Response(c, result)
}

func GetCommonFollows(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	idStr := c.Param("id")
	if idStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "查询关注ID不能为空")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "查询关注ID格式不正确")
	}
	result := service.GetCommonFollows(c.Request.Context(), userID.(uint), uint(id))
	utils.Response(c, result)
}
