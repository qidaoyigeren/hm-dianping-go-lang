package dao

import (
	"context"
	"hm-dianping-go/models"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	FollowKeyPrefix = "follow:"
)

func CreateFollow(ctx context.Context, follow *models.Follow) error {
	return DB.WithContext(ctx).Model(&models.Follow{}).Create(follow).Error
}

func SetFollowing(ctx context.Context, rds *redis.Client, userID uint, id uint) error {
	return rds.SAdd(ctx, FollowKeyPrefix+strconv.Itoa(int(userID)), strconv.Itoa(int(id))).Err()
}

func RemoveFollow(ctx context.Context, follow *models.Follow) error {
	return DB.WithContext(ctx).Model(&models.Follow{}).Delete(follow).Error
}

func DeleteFollowing(ctx context.Context, rds *redis.Client, userID uint, id uint) error {
	return rds.SRem(ctx, FollowKeyPrefix+strconv.Itoa(int(userID)), strconv.Itoa(int(id))).Err()
}

func GetCommonFollows(ctx context.Context, rds *redis.Client, userID uint, id uint) ([]uint, error) {
	var commonList []uint
	err := rds.
		SInter(ctx, FollowKeyPrefix+strconv.Itoa(int(userID)), FollowKeyPrefix+strconv.Itoa(int(id))).
		ScanSlice(&commonList)
	if err != nil {
		return nil, err
	}
	return commonList, nil
}

func GetUsersByIDs(ctx context.Context, userIDs []uint) ([]models.User, error) {
	var users []models.User
	err := DB.WithContext(ctx).Model(&models.User{}).Where("id IN (?)", userIDs).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
