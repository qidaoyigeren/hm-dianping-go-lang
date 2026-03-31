package service

import (
	"context"
	"fmt"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/utils"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func GetShopList(page int, size int) *utils.Result {
	var shops []models.Shop
	var total int64
	offset := (page - 1) * size
	//获取总数
	dao.DB.Model(&models.Shop{}).Count(&total)
	err := dao.DB.Offset(offset).Limit(size).Find(&shops).Error
	if err != nil {
		return utils.ErrorResult("查询失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"list":  shops,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
func GetShopById(ctx context.Context, id uint) *utils.Result {
	//布隆过滤器检查，防止缓存击穿
	exists, err := utils.CheckIDExistsWithRedis(ctx, dao.Redis, "shop", id)
	if err != nil {
		log.Printf("[BloomFilter] 检查商铺ID %d 时发生错误: %v，降级放行", id, err)
	} else if exists == false {
		return utils.ErrorResult("商铺不存在")
	}
	//先从缓存查
	shop, err := dao.GetShopCacheById(ctx, dao.Redis, id)
	if err == nil && shop != nil {
		return utils.SuccessResultWithData(shop)
	}
	//缓存未命中，使用互斥锁防止缓存击穿
	lockKey := fmt.Sprintf("lock:shop:%d", id)
	// 使用分布式锁（带自动续期）
	lock := utils.NewDistributedLock(dao.Redis, lockKey, 30*time.Second)
	//尝试获取锁
	if !lock.TryLock(ctx) {
		time.Sleep(50 * time.Millisecond)
		shop, err := dao.GetShopCacheById(ctx, dao.Redis, id)
		if err == nil && shop != nil {
			return utils.SuccessResultWithData(shop)
		}
		// 如果缓存仍然没有数据，返回错误
		return utils.ErrorResult("服务繁忙，请稍后重试")
	}
	//获取锁成功，确保释放锁
	defer lock.UnLock(ctx)
	//再次检查缓存（双重检查锁定模式）
	shop, err = dao.GetShopCacheById(ctx, dao.Redis, id)
	if err == nil && shop != nil {
		return utils.SuccessResultWithData(shop)
	}
	//查询数据库
	shop, err = dao.GetShopById(ctx, dao.DB, id)
	if err != nil {
		return utils.ErrorResult("查询失败: " + err.Error())
	}
	//设置缓存
	err = dao.SetShopCacheById(ctx, dao.Redis, id, shop)
	if err != nil {
		return utils.ErrorResult("设置缓存失败: " + err.Error())
	}
	//返回
	return utils.SuccessResultWithData(shop)
}

func GetShopByType(typeId uint, page, size int) *utils.Result {
	var shops []models.Shop
	var total int64

	offset := (page - 1) * size

	// 获取总数
	dao.DB.Model(&models.Shop{}).Where("type_id = ?", typeId).Count(&total)

	// 分页查询
	err := dao.DB.Where("type_id = ?", typeId).Offset(offset).Limit(size).Find(&shops).Error
	if err != nil {
		return utils.ErrorResult("查询失败")
	}

	return utils.SuccessResultWithData(map[string]interface{}{
		"list":  shops,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func GetShopByName(name string, page int, size int) *utils.Result {
	var shops []models.Shop
	var total int64
	offset := (page - 1) * size
	// 获取总数
	dao.DB.Model(&models.Shop{}).Where("name like ?", "%"+name+"%").Count(&total)
	// 分页查询
	err := dao.DB.Where("name like ?", "%"+name+"%").Offset(offset).Limit(size).Find(&shops).Error
	if err != nil {
		return utils.ErrorResult("查询失败")
	}

	return utils.SuccessResultWithData(map[string]interface{}{
		"list":  shops,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func SaveShop(ctx context.Context, m *models.Shop) *utils.Result {
	// 1. 参数校验
	if m.Name == "" || m.Address == "" || m.TypeID == 0 {
		return utils.ErrorResult("名称，地址与类型是必填")
	}
	// 2. 启动事务
	tx := dao.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 3. 创建商铺
	if err := dao.CreateShop(ctx, tx, m); err != nil {
		tx.Rollback()
		return utils.ErrorResult("创建商铺失败")
	}
	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.ErrorResult("提交事务失败")
	}
	// 5. 将新商铺ID添加到布隆过滤器（异步，不阻塞主流程）
	go func() {
		bgctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		bf := utils.CreateShopBloomFilter(dao.Redis)
		if _, err := bf.AddID(bgctx, m.ID); err != nil {
			log.Printf("[BloomFilter] 添加商铺ID %d 到布隆过滤器时发生错误: %v", m.ID, err)
		}
	}()
	// 6. 更新地理位置缓存（如果坐标有效）
	if m.X != 0 && m.Y != 0 {
		go func() {
			bgctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			key := dao.ShopLocationCache + strconv.Itoa(int(m.TypeID))
			if err := dao.Redis.GeoAdd(bgctx, key, &redis.GeoLocation{
				Name:      strconv.Itoa(int(m.ID)),
				Longitude: m.X,
				Latitude:  m.Y,
			}).Err(); err != nil {
				log.Printf("[Geo] 添加商铺地理位置缓存失败: %v", err)
			}
		}()
	}
	return utils.SuccessResultWithData(m)
}

func UpdateShopById(ctx context.Context, shop *models.Shop) *utils.Result {
	//启动事务
	tx := dao.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	//更新数据库
	if err := dao.UpdateShop(ctx, tx, shop); err != nil {
		tx.Rollback()
		return utils.ErrorResult("更新失败")
	}
	//提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.ErrorResult("提交事务失败")
	}
	//事务成功后删除缓存
	err := dao.DelShopCacheById(ctx, dao.Redis, shop.ID)
	if err != nil {
		log.Printf("警告: 删除缓存失败，商铺ID=%d, 错误=%v", shop.ID, err)
	}
	//返回结果
	return utils.SuccessResult("更新成功")
}

// GetNearbyShops 获取某个店铺的附近某个距离的所有点
func GetNearbyShops(ctx context.Context, shopId uint, radius float64, count int) *utils.Result {
	// 1. 查询店铺
	shop, err := dao.GetShopById(ctx, dao.DB, shopId)
	if err != nil {
		return utils.ErrorResult("查询店铺失败: " + err.Error())
	}

	// 2. 查询附近的同类型商铺
	shopIds, err := dao.GetNearbyShops(ctx, dao.Redis, shop, radius, "km", count)
	if err != nil {
		return utils.ErrorResult("查询附近商铺失败: " + err.Error())
	}

	// 3. 返回结果
	return utils.SuccessResultWithData(shopIds)
}
