package audit

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/mysunshines/blog-user/internal/model"
	user "github.com/mysunshines/blog-user/proto/pb"
)

// ActionToShort 将 proto 枚举 AuditAction 转为落库/展示用的短动作字符串。
// 例如 AUDIT_ACTION_UPDATE_USER -> "update_user"，AUDIT_ACTION_COMMENT_CREATE -> "comment_create"。
// 落库存短名便于管理端按 action 过滤与展示，同时保持上报方强类型。
func ActionToShort(a user.AuditAction) string {
	return strings.ToLower(strings.TrimPrefix(a.String(), "AUDIT_ACTION_"))
}

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
