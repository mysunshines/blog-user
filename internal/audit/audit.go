package audit

import (
	"context"

	"gorm.io/gorm"

	"github.com/mysunshines/blog-user/internal/model"
)

// Record 将一条操作日志写入本地 operation_logs 表。
// 各业务服务（包括 user-service 自身与通过 gRPC 上报的 article-service）
// 最终都由 user-service 统一落库。
func Record(ctx context.Context, db *gorm.DB, log *model.OperationLog) error {
	return db.WithContext(ctx).Create(log).Error
}

// List 分页查询操作日志，支持按 action / target_type / operator_id 过滤。
func List(ctx context.Context, db *gorm.DB, page, pageSize int, action, targetType string, operatorID uint) ([]model.OperationLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := db.WithContext(ctx).Model(&model.OperationLog{})
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if targetType != "" {
		q = q.Where("target_type = ?", targetType)
	}
	if operatorID != 0 {
		q = q.Where("operator_id = ?", operatorID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.OperationLog
	if err := q.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
