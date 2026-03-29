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
