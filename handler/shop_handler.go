package handler

import (
	"hm-dianping-go/models"
	"hm-dianping-go/service"
	"hm-dianping-go/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetShopList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	result := service.GetShopList(page, size)
	utils.Response(c, result)
}
func GetShopById(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的商铺ID")
		return
	}
	result := service.GetShopById(c.Request.Context(), uint(id))
	utils.Response(c, result)
}

func GetShopByType(c *gin.Context) {
	typeIdStr := c.Query("typeId")
	if typeIdStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "类型ID不能为空")
		return
	}
	typeId, err := strconv.ParseUint(typeIdStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "类型转换失败")
	}
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	// 防御性判断：如果转换出来是 0，强制改为默认值
	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	result := service.GetShopByType(uint(typeId), current, size)
	utils.Response(c, result)
}

func GetShopByName(c *gin.Context) {
	name := c.Query("name")
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	// 防御性判断：如果转换出来是 0，强制改为默认值
	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	result := service.GetShopByName(name, current, size)
	utils.Response(c, result)
}

func SaveShop(c *gin.Context) {
	var shop models.Shop
	if err := c.ShouldBindJSON(&shop); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数绑定失败")
		return
	}
	result := service.SaveShop(c.Request.Context(), &shop)
	utils.Response(c, result)
}

func UpdateShop(c *gin.Context) {
	var shop models.Shop
	if err := c.ShouldBindJSON(&shop); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数绑定失败")
		return
	}
	result := service.UpdateShopById(c.Request.Context(), &shop)
	utils.Response(c, result)
}

// GetNearbyShops 获取某个店铺的附近某个距离的所有点
func GetNearbyShops(c *gin.Context) {
	// 1. 参数校验
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的商铺ID")
		return
	}

	radius, err := strconv.ParseFloat(c.DefaultQuery("radius", "1.0"), 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的半径")
		return
	}

	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的数量")
		return
	}

	// 2. 查询附近的商铺
	result := service.GetNearbyShops(c.Request.Context(), uint(id), radius, count)
	utils.Response(c, result)
}
