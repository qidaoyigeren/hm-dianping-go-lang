package dao

import (
	"context"
	"fmt"
	"time"
)

const (
	// LoginCodePrefix 登录验证码Redis key前缀
	LoginCodePrefix = "dianping:user:login:phone:"
	// DefaultCodeExpiration 默认验证码过期时间（5分钟）
	DefaultCodeExpiration = 5 * time.Minute
)

func CheckLoginCodeExists(phone string) (bool, error) {
	//首先检查redis是否存在，不存在直接返回
	if Redis == nil {
		return false, fmt.Errorf("redis is not initialized")
	}
	//控制数据库操作超时
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	//通过key在redis查找
	key := LoginCodePrefix + phone
	exists, err := Redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check login code existence: %v", err)
	}
	//返回是否查询得到
	return exists > 0, nil
}

func GetLoginCodeExpireTime(phone string) (time.Duration, error) {
	//首先检查redis是否存在，不存在直接返回
	if Redis == nil {
		return 0, fmt.Errorf("redis is not initialized")
	}
	//控制数据库操作超时
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	//通过key在redis查找
	key := LoginCodePrefix + phone
	expiration, err := Redis.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get login code expiration time: %v", err)
	}
	return expiration, nil
}
func SetLoginCode(phone, code string, duration time.Duration) error {
	if Redis == nil {
		return fmt.Errorf("redis is not initialized")
	}
	if duration <= 0 {
		duration = DefaultCodeExpiration
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	key := LoginCodePrefix + phone
	err := Redis.Set(ctx, key, code, duration).Err()
	if err != nil {
		return fmt.Errorf("failed to set login code: %v", err)
	}
	return nil
}
