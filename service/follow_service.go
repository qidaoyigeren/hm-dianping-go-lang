package service

import (
	"context"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/utils"
)

func Follow(ctx context.Context, userID uint, id uint, isFollow bool) *utils.Result {
	if isFollow {
		//数据库添加follow信息
		//follow信息
		follow := &models.Follow{
			UserID:       userID,
			FollowUserID: id,
		}
		if err := dao.CreateFollow(ctx, follow); err != nil {
			return utils.ErrorResult("关注失败")
		}
		//redis存储关注信息
		if err := dao.SetFollowing(ctx, dao.Redis, userID, id); err != nil {
			return utils.ErrorResult("关注失败")
		}
		return utils.SuccessResult("关注成功")
	} else {
		var follow models.Follow
		err := dao.DB.WithContext(ctx).Where("user_id = ? AND follow_user_id = ?", userID, id).First(&follow).Error
		if err != nil {
			return utils.ErrorResult("获取关注失败")
		}
		if err := dao.RemoveFollow(ctx, &follow); err != nil {
			return utils.ErrorResult("取消关注失败")
		}
		if err := dao.DeleteFollowing(ctx, dao.Redis, userID, id); err != nil {
			return utils.ErrorResult("取消关注失败")
		}
		return utils.SuccessResult("取消关注成功")
	}
}

func IsFollow(ctx context.Context, userID uint, id uint) *utils.Result {
	var count int64
	err := dao.DB.WithContext(ctx).Model(&models.Follow{}).Where("user_id = ? AND follow_user_id = ?", userID, id).Count(&count).Error
	if err != nil {
		return utils.ErrorResult("获取关注失败")
	}
	return utils.SuccessResultWithData(count > 0)
}

func GetCommonFollows(ctx context.Context, userID uint, id uint) *utils.Result {
	//使用redis查询共同关注的用户ID
	commonFollowIDs, err := dao.GetCommonFollows(ctx, dao.Redis, userID, id)
	if err != nil {
		return utils.ErrorResult("获取共同关注失败")
	}
	//判断有无共同关注
	if len(commonFollowIDs) == 0 {
		return utils.SuccessResult("无共同关注")
	}
	//根据ID查询用户信息
	users, err := dao.GetUsersByIDs(ctx, commonFollowIDs)
	if err != nil {
		return utils.ErrorResult("获取用户信息失败")
	}
	//返回
	return utils.SuccessResultWithData(users)
}
