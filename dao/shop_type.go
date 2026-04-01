package dao

import (
	"context"
	"encoding/json"
	"hm-dianping-go/models"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	ShopTypeCache = "cache:shop_type"
)

func GetShopTypeListCache(ctx context.Context, redis *redis.Client) ([]*models.ShopType, error) {
	var shopTypeList []*models.ShopType
	res := redis.Get(ctx, ShopTypeCache)
	if res.Err() != nil {
		return nil, res.Err()
	}
	if err := json.Unmarshal([]byte(res.Val()), &shopTypeList); err != nil {
		return nil, err
	}
	return shopTypeList, nil
}

func SetShopTypeListCache(ctx context.Context, rds *redis.Client, list []*models.ShopType) error {
	listStr, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return rds.Set(ctx, ShopTypeCache, listStr, time.Hour).Err()
}

func GetShopTypeListDB(ctx context.Context, db *gorm.DB) ([]*models.ShopType, error) {
	var shopTypeList []*models.ShopType
	err := db.WithContext(ctx).Model(&models.ShopType{}).Order("sort").Find(&shopTypeList).Error
	if err != nil {
		return nil, err
	}
	return shopTypeList, nil
}
