package service

import (
	"context"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/utils"
	"time"

	"github.com/gin-gonic/gin"
)

func GetVoucherList(c *gin.Context, u uint32) *utils.Result {
	var voucherList []models.Voucher
	err := dao.DB.WithContext(c).Where("shop_id = ? and status = 1", u).Find(&voucherList).Error
	if err != nil {
		return utils.ErrorResult("获取优惠券列表失败")
	}
	return utils.SuccessResultWithData(voucherList)
}

type AddVoucherReq struct {
	ShopId      uint32    `json:"shopId" binding:"required"`
	Title       string    `json:"title" binding:"required"`
	SubTitle    string    `json:"subTitle"`
	Rules       string    `json:"rules"`
	PayValue    int64     `json:"payValue" binding:"required"`
	ActualValue int64     `json:"actualValue" binding:"required"`
	BeginTime   time.Time `json:"beginTime" binding:"required"`
	EndTime     time.Time `json:"endTime" binding:"required"`
}

func AddVoucher(c context.Context, s *AddVoucherReq) *utils.Result {
	if s.EndTime.Before(s.BeginTime) {
		return utils.ErrorResult("结束时间不能早于开始时间")
	}
	if s.BeginTime.After(s.EndTime) {
		return utils.ErrorResult("结束时间不能早于开始时间")
	}
	if s.PayValue < 0 {
		return utils.ErrorResult("支付金额不能小于0")
	}
	voucher := models.Voucher{
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
		ShopID:      uint(s.ShopId),
		Title:       s.Title,
		SubTitle:    s.SubTitle,
		Rules:       s.Rules,
		PayValue:    s.PayValue,
		ActualValue: s.ActualValue,
		Type:        0,
		Status:      1,
		Stock:       0,
		BeginTime:   &s.BeginTime,
		EndTime:     &s.EndTime,
	}
	if err := dao.DB.WithContext(c).Create(&voucher).Error; err != nil {
		return utils.ErrorResult("添加优惠券失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"voucherId": voucher.ID,
		"message":   "普通优惠券创建成功",
	})
}

type AddSeckillVoucherReq struct {
	ShopID      uint      `json:"shopId" binding:"required"`
	Title       string    `json:"title" binding:"required"`
	SubTitle    string    `json:"subTitle"`
	Rules       string    `json:"rules"`
	PayValue    int64     `json:"payValue" binding:"required"`
	ActualValue int64     `json:"actualValue" binding:"required"`
	Stock       int       `json:"stock" binding:"required,min=1"`
	BeginTime   time.Time `json:"beginTime" binding:"required"`
	EndTime     time.Time `json:"endTime" binding:"required"`
}

func AddSeckillVoucher(ctx context.Context, req *AddSeckillVoucherReq) *utils.Result {
	if req.EndTime.Before(req.BeginTime) {
		return utils.ErrorResult("结束时间不能早于开始时间")
	}
	if req.BeginTime.After(req.EndTime) {
		return utils.ErrorResult("结束时间不能早于开始时间")
	}
	if req.PayValue < 0 {
		return utils.ErrorResult("支付金额不能小于0")
	}
	tx := dao.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return utils.ErrorResult("开启事务失败")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	voucher := models.Voucher{
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
		ShopID:      req.ShopID,
		Title:       req.Title,
		SubTitle:    req.SubTitle,
		Rules:       req.Rules,
		PayValue:    req.PayValue,
		ActualValue: req.ActualValue,
		Type:        1,
		Status:      1,
		BeginTime:   &req.BeginTime,
		EndTime:     &req.EndTime,
	}
	if err := tx.Create(&voucher).Error; err != nil {
		tx.Rollback()
		return utils.ErrorResult("添加优惠券失败")
	}
	seckillVoucher := &models.SeckillVoucher{
		VoucherID:  voucher.ID,
		Stock:      req.Stock,
		CreateTime: time.Now(),
		BeginTime:  req.BeginTime,
		EndTime:    req.EndTime,
		UpdateTime: time.Now(),
	}
	if err := tx.Create(seckillVoucher).Error; err != nil {
		tx.Rollback()
		return utils.ErrorResult("添加秒杀优惠券失败")
	}
	//创建秒杀券缓存
	if err := dao.SetSeckillVoucheherStockCache(ctx, dao.Redis, voucher.ID, req.Stock); err != nil {
		tx.Rollback()
		return utils.ErrorResult("添加秒杀券缓存失败")
	}
	if err := tx.Commit().Error; err != nil {
		return utils.ErrorResult("提交事务失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"voucherId": voucher.ID,
		"message":   "秒杀优惠券创建成功",
	})
}

func GetSeckillVoucher(ctx context.Context, id int) *utils.Result {
	var voucher models.Voucher
	if err := dao.DB.WithContext(ctx).Where("id = ?", id).First(&voucher).Error; err != nil {
		return utils.ErrorResult("获取优惠券失败")
	}
	if voucher.Type != 1 {
		return utils.ErrorResult("该优惠券不是秒杀券")
	}
	seckillVoucher, err := dao.GetSeckillVoucher(ctx, id)
	if err != nil {
		return utils.ErrorResult("获取秒杀券失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"voucher":        voucher,
		"seckillVoucher": seckillVoucher,
	})
}
