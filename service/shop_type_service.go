package service

import (
	"context"
	"hm-dianping-go/dao"
	"hm-dianping-go/utils"
)

func GetShopTypeList(ctx context.Context) *utils.Result {
	//先从缓存查
	list, err := dao.GetShopTypeListCache(ctx, dao.Redis)
	if err == nil {
		return utils.SuccessResultWithData(list)
	}
	//从数据库查
	list, err = dao.GetShopTypeListDB(ctx, dao.DB)
	if err != nil {
		return utils.ErrorResult("查询失败")
	}
	//存入到缓存
	err = dao.SetShopTypeListCache(ctx, dao.Redis, list)
	return utils.SuccessResultWithData(list)
}
