package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hm-dianping-go/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func GetShopById(ctx context.Context, db *gorm.DB, id uint) (*models.Shop, error) {
	shop := &models.Shop{}
	err := db.Where("id=?", id).First(shop).Error
	if err != nil {
		return nil, err
	}
	return shop, nil
}

/* ================缓存相关================ */

const (
	ShopCache         = "cache:shop:description:"
	ShopLocationCache = "cache:shop:location:"
)

func GetShopCacheById(ctx context.Context, rds *redis.Client, shopId uint) (*models.Shop, error) {
	//构造key
	key := ShopCache + strconv.Itoa(int(shopId))
	//redis查询key得到result
	result := rds.Get(ctx, key)
	//判断redis是否返回错误
	if result.Err() != nil {
		//区分缓存未命中和其他错误
		if errors.Is(result.Err(), redis.Nil) {
			return nil, errors.New("缓存不存在")
		} else {
			return nil, result.Err()
		}
	}
	//redis键存在，获取JSON字符串
	jsonStr, err := result.Result()
	if err != nil {
		return nil, err
	}
	//JSON反反序列化
	shop := &models.Shop{}
	err = json.Unmarshal([]byte(jsonStr), shop)
	if err != nil {
		return nil, fmt.Errorf("cache data unmarshal failed: %w", err)
	}
	//返回
	return shop, nil
}
func SetShopCacheById(ctx context.Context, rds *redis.Client, shopId uint, shop *models.Shop) error {
	jsonStr, err := json.Marshal(shop)
	if err != nil {
		return fmt.Errorf("cache data marshal failed: %w", err)
	}
	// 存储到Redis
	err = rds.Set(ctx, ShopCache+strconv.Itoa(int(shopId)), jsonStr, time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

func CreateShop(ctx context.Context, tx *gorm.DB, m *models.Shop) error {
	return DB.WithContext(ctx).Create(m).Error
}

func DelShopCacheById(ctx context.Context, rds *redis.Client, id uint) error {
	err := rds.Del(ctx, ShopCache+strconv.Itoa(int(id))).Err()
	if err != nil {
		return err
	}
	return nil
}

func UpdateShop(ctx context.Context, db *gorm.DB, shop *models.Shop) error {
	err := db.Model(&models.Shop{}).Where("id=?", shop.ID).Updates(shop).Error
	if err != nil {
		return err
	}
	return nil
}

// GetNearbyShops 获取某个店铺的附近某个距离的所有点
func GetNearbyShops(ctx context.Context, rds *redis.Client, shop *models.Shop, radius float64, unit string, count int) ([]uint, error) {
	key := ShopLocationCache + strconv.Itoa(int(shop.TypeID))
	result, err := rds.GeoSearch(ctx, key, &redis.GeoSearchQuery{
		Latitude:   shop.Y,
		Longitude:  shop.X,
		Radius:     radius,
		RadiusUnit: unit,
		Count:      count,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get geo cache: %w", err)
	}
	// 2. 解析结果，提取店铺ID
	var shopIds []uint
	for _, loc := range result {
		id, _ := strconv.Atoi(loc)
		shopIds = append(shopIds, uint(id))
	}
	return shopIds, nil
}
func GetAllShopIDs(ctx context.Context, db *gorm.DB) ([]uint, error) {
	var ids []uint
	err := db.WithContext(ctx).Model(&models.Shop{}).Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
func GetAllShopIDsWithContext(ctx context.Context) ([]uint, error) {
	return GetAllShopIDs(ctx, DB)
}

func LoadShopData(ctx context.Context, db *gorm.DB, rds *redis.Client) error {
	// 1. 查询所有的店铺
	var shops []models.Shop
	err := db.WithContext(ctx).Model(&models.Shop{}).Find(&shops).Error
	if err != nil {
		return fmt.Errorf("failed to query shops: %w", err)
	}

	// 2. 遍历店铺，根据类型进行缓存
	for _, shop := range shops {
		// 2.1 使用 GEOADD 存储店铺位置信息
		err = rds.GeoAdd(ctx, ShopLocationCache+strconv.Itoa(int(shop.TypeID)), &redis.GeoLocation{
			Name:      strconv.Itoa(int(shop.ID)),
			Latitude:  shop.Y,
			Longitude: shop.X,
		}).Err()

		if err != nil {
			return fmt.Errorf("failed to set geo cache: %w", err)
		}
	}

	return nil
}
