package handler

import (
	"hm-dianping-go/service"
	"hm-dianping-go/utils"

	"github.com/gin-gonic/gin"
)

func GetShopTypeList(c *gin.Context) {
	result := service.GetShopTypeList(c.Request.Context())
	utils.Response(c, result)
}
