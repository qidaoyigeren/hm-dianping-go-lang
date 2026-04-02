package dao

import (
	"context"
	"hm-dianping-go/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func CreateBlog(ctx context.Context, m *models.Blog) error {
	return DB.WithContext(ctx).Create(m).Error
}

func GetFollowersByUserID(ctx context.Context, userID uint) ([]models.User, error) {
	var users []models.User
	err := DB.WithContext(ctx).Table("tb_follow").
		Select("u.*").
		Joins("JOIN tb_user u ON tb_follow.user_id = u.id").
		Where("tb_follow.follow_user_id = ?", userID).
		Find(&users).Error

	return users, err
}

const (
	// 博客点赞集合的键名格式：blog_like:%d
	blogLikeKey = "blog:liked:"
	feedKey     = "feed:"
)

func FeedToUserRedis(ctx context.Context, rds *redis.Client, userID uint, blogID uint) error {
	return rds.ZAdd(ctx, feedKey+strconv.Itoa(int(userID)), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: strconv.Itoa(int(blogID)),
	}).Err()
}

func IsLikedMember(ctx context.Context, rds *redis.Client, userID uint, blogID uint) (bool, error) {
	_, err := rds.ZScore(ctx, blogLikeKey+strconv.Itoa(int(blogID)), strconv.Itoa(int(userID))).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func RemoveLikedMember(ctx context.Context, rds *redis.Client, userID uint, blogID uint) error {
	return rds.ZRem(ctx, blogLikeKey+strconv.Itoa(int(blogID)), strconv.Itoa(int(userID))).Err()
}

func DecrementBlogLiked(ctx context.Context, blogID uint) error {
	return DB.WithContext(ctx).Model(&models.Blog{}).Where("id = ?", blogID).UpdateColumn("liked", gorm.Expr("liked - 1")).Error
}

func AddLikedMember(ctx context.Context, rds *redis.Client, userID uint, blogID uint) error {
	return rds.ZAdd(ctx, blogLikeKey+strconv.Itoa(int(blogID)), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: strconv.Itoa(int(userID)),
	}).Err()
}

func IncrementBlogLiked(ctx context.Context, blogID uint) error {
	return DB.WithContext(ctx).Model(&models.Blog{}).Where("id = ?", blogID).UpdateColumn("liked", gorm.Expr("liked + 1")).Error
}

func GetHotBlogList(ctx context.Context, userID uint, offset int, size int) ([]models.Blog, int64, error) {
	var blogs []models.Blog
	var total int64
	err := DB.WithContext(ctx).Model(&models.Blog{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DB.WithContext(ctx).Model(&models.Blog{}).Order("liked desc, created_at desc").Offset(offset).Limit(size).Find(&blogs).Error
	return blogs, total, err
}

func GetBlogListOfUser(ctx context.Context, userID uint, offset int, size int) ([]models.Blog, int64, error) {
	var blogs []models.Blog
	var total int64
	err := DB.WithContext(ctx).Model(&models.Blog{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DB.WithContext(ctx).Model(&models.Blog{}).Where("user_id = ?", userID).Order("created_at desc").Offset(offset).Limit(size).Find(&blogs).Error
	return blogs, total, err
}

func GetBlogById(ctx context.Context, id uint) (models.Blog, error) {
	var blog models.Blog
	err := DB.WithContext(ctx).Where("id = ?", id).First(&blog).Error
	return blog, err
}

func GetFeedFromUserRedis(ctx context.Context, rds *redis.Client, userID uint, lastID uint, offset int, count int) ([]uint, uint, int, error) {
	result, err := rds.ZRevRangeByScoreWithScores(ctx, feedKey+strconv.Itoa(int(userID)), &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.Itoa(int(lastID)),
		Offset: int64(offset),
		Count:  int64(count),
	}).Result()
	if err != nil {
		return nil, 0, 0, err
	}
	if len(result) == 0 {
		return nil, 0, 0, nil
	}
	var blogIDs []uint
	minTime := result[len(result)-1].Score
	offset = 0
	for _, v := range result {
		blogID, _ := strconv.Atoi(v.Member.(string))
		blogIDs = append(blogIDs, uint(blogID))
		if v.Score == minTime {
			offset++
		}
	}
	return blogIDs, uint(minTime), offset, err
}

func GetBlogsByIds(ctx context.Context, blogIDs []uint) ([]models.Blog, error) {
	var blogs []models.Blog
	err := DB.WithContext(ctx).Where("id IN ?", blogIDs).Find(&blogs).Error
	return blogs, err
}
