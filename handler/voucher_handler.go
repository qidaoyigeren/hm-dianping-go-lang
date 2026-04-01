package handler

import (
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetVoucherList(c *gin.Context) {
	shopIdStr := c.Param("shopId")
	if shopIdStr == "" {
		utils.ErrorResult("shopId不能为空")
		return
	}
	shopId, err := strconv.ParseUint(shopIdStr, 10, 32)
	if err != nil {
		utils.ErrorResult("shopId转换失败")
		return
	}
	result := service.GetVoucherList(c, uint32(shopId))
	utils.Response(c, result)

}

func AddVoucher(c *gin.Context) {
	var req service.AddVoucherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误")
		return
	}
	result := service.AddVoucher(c.Request.Context(), &req)
	utils.Response(c, result)
}

func AddSeckillVoucher(c *gin.Context) {
	var req service.AddSeckillVoucherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误")
		return
	}
	result := service.AddSeckillVoucher(c.Request.Context(), &req)
	utils.Response(c, result)
}

func GetSeckillVoucher(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "id不能为空")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的优惠券id")
	}
	result := service.GetSeckillVoucher(c.Request.Context(), id)
	utils.Response(c, result)
}
