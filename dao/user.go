package dao

import (
	"context"
	"hm-dianping-go/models"
)

func CheckUserExistsByPhone(phone string) (bool, error) {
	var count int64
	err := DB.Model(&models.User{}).Where("phone = ?", phone).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateUser(user *models.User) error {
	return DB.Create(user).Error
}
func GetUserByPhone(phone string) (*models.User, error) {
	var user *models.User
	err := DB.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil

}
func GetUserById(userID uint) (*models.User, error) {
	var user models.User
	err := DB.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func UpdateUser(user *models.User) error {
	return DB.Save(user).Error
}
func GetAllUserIDs() ([]uint, error) {
	var ids []uint
	err := DB.Model(&models.User{}).Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
func GetAllUserIDsWithContext(ctx context.Context) ([]uint, error) {
	return GetAllUserIDs()
}
