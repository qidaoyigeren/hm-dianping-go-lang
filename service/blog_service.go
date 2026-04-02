package service

import (
	"context"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/utils"
)

func CreateBlog(ctx context.Context, userID uint, shopID uint, title string, content string, images string) *utils.Result {
	blog := models.Blog{
		UserID:  userID,
		Title:   title,
		Content: content,
		Images:  images,
		ShopID:  shopID,
	}
	if err := dao.CreateBlog(ctx, &blog); err != nil {
		return utils.ErrorResult("创建失败")
	}
	followers, err := dao.GetFollowersByUserID(ctx, userID)
	if err != nil {
		return utils.ErrorResult("获取关注列表失败")
	}
	// 遍历，然后将内容推送到关注用户的队列中
	for _, follower := range followers {
		if err := dao.FeedToUserRedis(ctx, dao.Redis, follower.ID, blog.ID); err != nil {
			return utils.ErrorResult("发布博客失败")
		}
	}

	return utils.SuccessResultWithData(blog.ID)
}

func LikeBlog(ctx context.Context, userID uint, blogID uint) *utils.Result {
	liked, err := dao.IsLikedMember(ctx, dao.Redis, userID, blogID)
	if err != nil {
		return utils.ErrorResult("点赞失败")
	}
	if liked {
		if err := dao.RemoveLikedMember(ctx, dao.Redis, userID, blogID); err != nil {
			return utils.ErrorResult("取消点赞失败")
		}
		if err := dao.DecrementBlogLiked(ctx, blogID); err != nil {
			return utils.ErrorResult("减少点赞数失败")
		}
		return utils.SuccessResult("取消点赞成功")
	} else {
		if err := dao.AddLikedMember(ctx, dao.Redis, userID, blogID); err != nil {
			return utils.ErrorResult("点赞失败")
		}
		if err := dao.IncrementBlogLiked(ctx, blogID); err != nil {
			return utils.ErrorResult("增加点赞数失败")
		}
		return utils.SuccessResult("点赞成功")
	}
}

func GetHotBlogList(ctx context.Context, userID uint, page int, size int) *utils.Result {
	offset := (page - 1) * size
	blogs, total, err := dao.GetHotBlogList(ctx, userID, offset, size)
	if err != nil {
		return utils.ErrorResult("获取热门博客列表失败")
	}
	for _, blog := range blogs {
		if err := isBlogLiked(ctx, &blog, userID); err != nil {
			return utils.ErrorResult("获取博客点赞状态失败")
		}
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"list":  blogs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
func isBlogLiked(ctx context.Context, blog *models.Blog, userID uint) error {
	liked, err := dao.IsLikedMember(ctx, dao.Redis, userID, blog.ID)
	if liked {
		blog.IsLiked = true
	} else {
		blog.IsLiked = false
	}
	return err
}

func GetBlogListOfUser(ctx context.Context, userID uint, page int, size int) *utils.Result {
	offset := (page - 1) * size
	blogs, total, err := dao.GetBlogListOfUser(ctx, userID, offset, size)
	if err != nil {
		return utils.ErrorResult("获取用户博客列表失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"list":  blogs,
		"total": total,
		"page":  page,
		"size":  size,
	})

}

func GetBlogById(ctx context.Context, blogID uint, userId uint) *utils.Result {
	blog, err := dao.GetBlogById(ctx, blogID)
	if err != nil {
		return utils.ErrorResult("获取博客失败")
	}
	if err := isBlogLiked(ctx, &blog, userId); err != nil {
		return utils.ErrorResult("获取博客点赞状态失败")
	}
	return utils.SuccessResultWithData(blog)
}

func GetBlogListOfFollow(ctx context.Context, userID uint, lastID uint, offset int, count int) *utils.Result {
	//从关注用户队列获取博客ID
	blogIDs, minTime, offset, err := dao.GetFeedFromUserRedis(ctx, dao.Redis, userID, lastID, offset, count)
	if err != nil {
		return utils.ErrorResult("获取关注用户队列失败")
	}
	//根据博客ID获取博客详情
	blogs, err := dao.GetBlogsByIds(ctx, blogIDs)
	//检查是否点赞
	for _, blog := range blogs {
		if err := isBlogLiked(ctx, &blog, userID); err != nil {
			return utils.ErrorResult("获取博客点赞状态失败")
		}
	}
	//封装结果
	return utils.SuccessResultWithData(map[string]interface{}{
		"list":   blogs,
		"minId":  minTime,
		"offset": offset,
	})
}
