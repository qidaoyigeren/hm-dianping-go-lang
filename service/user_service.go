package service

import (
	"fmt"
	"hm-dianping-go/dao"
	"hm-dianping-go/utils"
	"time"
)

func SendCode(phone string) *utils.Result {
	// 检查是否已存在未过期的验证码
	exists, err := dao.CheckLoginCodeExists(phone)
	// 防止频繁请求，进行避免
	if err != nil {
		return utils.ErrorResult("系统错误，请稍后重试")
	}
	if exists {
		// 获取剩余时间
		expireTime, _ := dao.GetLoginCodeExpireTime(phone)
		if expireTime > 0 {
			return utils.ErrorResult("验证码已发送，请勿频繁请求")
		}
	}
	// 生成6位随机验证码
	code := utils.GenerateRandomCode(6)
	// 将验证码存储到Redis，设置5分钟过期
	err = dao.SetLoginCode(phone, code, 5*time.Minute)
	if err != nil {
		return utils.ErrorResult("验证码发送失败，请稍后重试")
	}
	// TODO: 实现发送短信验证码功能
	// 这里可以集成短信服务商API，如阿里云短信、腾讯云短信等
	// 暂时在日志中输出验证码（仅用于开发测试）
	fmt.Printf("验证码：%s\n", code)
	return utils.SuccessResult("验证码发送成功")
}
