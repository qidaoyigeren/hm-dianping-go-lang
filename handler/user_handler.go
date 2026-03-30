package handler

import (
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
)

func SendCode(c *gin.Context) {
	phone := c.Query("phone")
	logger.Debug(phone)
	if phone == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "手机号不能为空")
		return
	}

	if !utils.IsPhoneValid(phone) {
		utils.ErrorResponse(c, http.StatusBadRequest, "手机号格式不正确")
		return
	}
	result := service.SendCode(phone)
	utils.Response(c, result)
}
func UserRegister(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		NickName string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	result := service.UserRegister(req.Phone, req.Code, req.Password, req.NickName)
	utils.Response(c, result)
}
func UserLogin(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	result := service.UserLogin(req.Phone, req.Code)
	utils.Response(c, result)
}
func UserLogout(c *gin.Context) {
	/// 从上下文中获取用户ID（由中间件设置）
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户未登录")
		return
	}
	// 获取 token
	token := c.GetHeader("Authorization")
	if token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "token 不能为空")
		return
	}
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, "token 格式不正确")
		return
	}
	// 调用 service 层处理登出
	result := service.UserLogout(userID.(uint), token)
	utils.Response(c, result)
}
func GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户未登录")
		return
	}
	result := service.GetUserInfo(userID.(uint))
	utils.Response(c, result)
}
func UpdateUserInfo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户未登录")
		return
	}
	var req struct {
		NickName string `json:"nickname"`
		Icon     string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误:"+err.Error())
		return
	}
	result := service.UpdateUserInfo(userID.(uint), req.NickName, req.Icon)
	utils.Response(c, result)
}
