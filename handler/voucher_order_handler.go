package handler

import (
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func SeckillVoucher(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	voucherIdStr := c.Param("voucherId")
	voucherId, err := strconv.ParseUint(voucherIdStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的优惠券ID")
		return
	}
	result := service.SeckillVoucher(c.Request.Context(), userID.(uint), uint(voucherId))
	utils.Response(c, result)
}
