package dao

import (
	"context"
	"encoding/json"
	"hm-dianping-go/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	SeckillVoucherCache = "cache:seckill_voucher:stock:"
)

func SetSeckillVoucheherStockCache(c context.Context, rds *redis.Client, id uint, stock int) error {
	key := SeckillVoucherCache + strconv.Itoa(int(id))
	data, err := json.Marshal(stock)
	if err != nil {
		return err
	}
	return rds.Set(c, key, data, time.Hour).Err()
}

func GetSeckillVoucher(ctx context.Context, id int) (*models.SeckillVoucher, error) {
	var seckillVoucher *models.SeckillVoucher
	err := DB.WithContext(ctx).Where("voucher_id=?", id).First(&seckillVoucher).Error
	if err != nil {
		return nil, err
	}
	return seckillVoucher, nil
}
