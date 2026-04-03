package service

import (
	"context"
	"fmt"
	"hm-dianping-go/dao"
	"hm-dianping-go/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取指定日期UV
func GetDailtUV(ctx context.Context, date string) *utils.Result {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return utils.ErrorResult("日期格式错误，请使用YYYY-MM-DD格式")
	}
	uvKEY := fmt.Sprintf("uv:daily:%s", date)
	count, err := dao.Redis.PFCount(ctx, uvKEY).Result()
	if err != nil {
		return utils.ErrorResult("获取UV失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"date": date,
		"uv":   count,
	})
}

func GetTodayUV(c *gin.Context) *utils.Result {
	return GetDailtUV(c, time.Now().Format("2006-01-02"))
}
func GetUVRange(ctx context.Context, startTime, endTime string) *utils.Result {
	start, err := time.Parse("2006-01-02", startTime)
	if err != nil {
		return utils.ErrorResult("开始日期格式错误，请使用YYYY-MM-DD格式")
	}
	end, err := time.Parse("2006-01-02", endTime)
	if err != nil {
		return utils.ErrorResult("结束日期格式错误，请使用YYYY-MM-DD格式")
	}
	if end.Before(start) || end.Sub(start).Hours() > 24*30 {
		return utils.ErrorResult("结束时间要在开始时间的(0,30)天内")
	}
	var results []map[string]interface{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		uvKey := fmt.Sprintf("uv:daily:%s", d.Format("2006-01-02"))
		count, err := dao.Redis.PFCount(ctx, uvKey).Result()
		if err != nil {
			count = 0
		}
		results = append(results, map[string]interface{}{
			"date": d.Format("2006-01-02"),
			"uv":   count,
		})
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"startDate": startTime,
		"endDate":   endTime,
		"data":      results,
	})
}

func GetRecentUV(ctx context.Context, days int) *utils.Result {
	if days <= 0 || days > 30 {
		return utils.ErrorResult("days参数必须在(0,30]范围内")
	}
	return GetUVRange(ctx, time.Now().AddDate(0, 0, -days).Format("2006-01-02"), time.Now().Format("2006-01-02"))
}
