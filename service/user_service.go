package service

import (
	"context"
	"fmt"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/utils"
	"time"
)

const (
	// LoginCodePrefix 登录验证码Redis key前缀
	LoginCodePrefix = "dianping:user:login:phone:"
	// DefaultCodeExpiration 默认验证码过期时间（5分钟）
	DefaultCodeExpiration = 5 * time.Minute
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
func UserRegister(phone, code, password, nickname string) *utils.Result {
	//校验手机号格式
	if utils.IsPhoneInvalid(phone) {
		return utils.ErrorResult("手机号格式不正确")
	}
	//校验验证码格式
	if utils.IsCodeInvalid(code) {
		return utils.ErrorResult("验证码格式不正确")
	}
	//校验密码格式
	if utils.IsPasswordInvalid(password) {
		return utils.ErrorResult("密码格式不正确")
	}
	//校验验证码
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	// 从Redis中获取验证码
	key := LoginCodePrefix + phone
	if dao.Redis == nil {
		return utils.ErrorResult("Redis连接失败")
	}
	val, err := dao.Redis.Get(ctx, key).Result()
	if err != nil {
		return utils.ErrorResult("验证码获取失败，请确认验证码是否存在或已经过期")
	}
	if val != code {
		return utils.ErrorResult("验证码错误")
	}
	//检验用户是否存在
	exists, err := dao.CheckUserExistsByPhone(phone)
	//不存在创建新用户
	if err != nil {
		return utils.ErrorResult("系统错误，请稍后重试")
	}
	if exists {
		return utils.ErrorResult("用户已存在")
	}
	npassword, err := utils.HashPassword(password)
	if err != nil {
		return utils.ErrorResult("密码加密失败")
	}
	user := models.User{
		Phone:    phone,
		Password: npassword,
		NickName: nickname,
	}
	if nickname == "" {
		user.NickName = "用户" + phone[len(phone)-4:]
	}
	if err := dao.CreateUser(&user); err != nil {
		return utils.ErrorResult("注册失败")
	}
	return utils.SuccessResult("注册成功")
}
func UserLogin(phone, code string) *utils.Result {
	//从Redis获取验证码
	storedCode, err := dao.GetLoginCode(phone)
	if err != nil {
		return utils.ErrorResult("验证码获取失败，请确认验证码是否存在或已经过期")
	}
	//验证验证码是否正确
	if storedCode != code {
		return utils.ErrorResult("验证码错误")
	}
	//验证成功后删除验证码
	_ = dao.DeleteLoginCode(phone)
	//根据手机号查询用户
	user, err := dao.GetUserByPhone(phone)
	if err != nil {
		//用户不存在，自动注册
		nuser := models.User{
			Phone:    phone,
			NickName: "用户" + phone[7:],
		}
		if err := dao.CreateUser(&nuser); err != nil {
			return utils.ErrorResult("登录失败")
		}
		user = &nuser
	}
	//生成JWT
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return utils.ErrorResult("用户登录失败")
	}
	//返回
	return utils.SuccessResultWithData(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"nickName": user.NickName,
			"phone":    user.Phone,
			"icon":     user.Icon,
		},
	})
}
func UserLogout(userID uint, token string) *utils.Result {
	claims, err := utils.ParseToken(token)
	if err != nil {
		return utils.ErrorResult("无效token")
	}
	expiresAt := claims.ExpiresAt.Time
	now := time.Now()
	if expiresAt.After(now) {
		ttl := expiresAt.Sub(now)
		if err := dao.AddTokenToBlacklist(token, ttl); err != nil {
			return utils.ErrorResult("登出失败，请稍后重试")
		}
	}
	return utils.SuccessResult("登出成功")
}
func GetUserInfo(userID uint) *utils.Result {
	user, err := dao.GetUserById(userID)
	if err != nil {
		return utils.ErrorResult("获取用户信息失败")
	}
	return utils.SuccessResultWithData(map[string]interface{}{
		"id":       user.ID,
		"phone":    user.Phone,
		"nickName": user.NickName,
		"icon":     user.Icon,
	})
}
func UpdateUserInfo(userID uint, nickname, icon string) *utils.Result {
	//获取用户
	user, err := dao.GetUserById(userID)
	if err != nil {
		return utils.ErrorResult("获取用户信息失败")
	}
	//修改
	if nickname != "" {
		user.NickName = nickname
	}
	if icon != "" {
		user.Icon = icon
	}
	//更新
	if err := dao.UpdateUser(user); err != nil {
		return utils.ErrorResult("更新用户信息失败")
	}
	return utils.SuccessResult("更新用户信息成功")
}
