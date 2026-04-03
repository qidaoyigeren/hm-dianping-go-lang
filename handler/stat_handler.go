package handler

import (
	"hm-dianping-go/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetTodayUV(c *gin.Context) {
	result := service.GetTodayUV(c)
	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusInternalServerError, result)
	}
}

func GetDailyUV(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "date is required",
		})
		return
	}
	result := service.GetDailtUV(c, date)
	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusInternalServerError, result)
	}
}

func GetRecentUV(c *gin.Context) {
	daysStr := c.Query("days")
	if daysStr == "" {
		daysStr = "7"
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "days is invalid",
		})
		return
	}
	result := service.GetRecentUV(c.Request.Context(), days)
	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, result)
	}
}

func GetUVRange(c *gin.Context) {
	startTime := c.Query("startDate")
	endTime := c.Query("endDate")
	if startTime == "" || endTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "startTime and endTime are required",
		})
		return
	}
	if endTime < startTime {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "endTime should be later than startTime",
		})
		return
	}
	result := service.GetUVRange(c.Request.Context(), startTime, endTime)
	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, result)
	}
}
