package dao

import (
	"context"
	"hm-dianping-go/models"

	"gorm.io/gorm"
)

func CreateVoucherOrder(ctx context.Context, db *gorm.DB, order *models.VoucherOrder) error {
	return db.WithContext(ctx).Create(order).Error
}
