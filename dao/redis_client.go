package dao

import (
	"context"
	"fmt"
	"hm-dianping-go/config"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// 全局数据库连接实例
var (
	Redis *redis.Client
)

const (
	LoginTokenBlackList = "dianping:user:blacklist:"
)

// InitRedis 初始化Redis连接
func InitRedis() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}
	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Redis.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	log.Println("Redis connected successfully")
	return nil
}

// CloseRedis 关闭Redis连接
func CloseRedis() error {
	if Redis != nil {
		return Redis.Close()
	}
	return nil
}

func AddTokenToBlacklist(token string, ttl time.Duration) error {
	ctx := context.Background()
	key := LoginTokenBlackList + token
	return Redis.Set(ctx, key, "1", ttl).Err()
}
func IsTokenInBlacklist(token string) (bool, error) {
	ctx := context.Background()
	key := LoginTokenBlackList + token
	exists, err := Redis.Exists(ctx, key).Result()
	return exists == 1, err
}
