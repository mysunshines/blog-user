package v1

import (
	"context"

	"github.com/mysunshines/blog-user/internal/audit"
	"github.com/mysunshines/blog-user/internal/model"
	user "github.com/mysunshines/blog-user/proto/pb"

	"gorm.io/gorm"
)

// GrpcAuditHandler 实现 user.v1.AuditService，作为各业务服务上报操作日志的汇聚点。
// article-service 等通过 gRPC 调用 RecordLog 写入；ListLogs 供本服务管理端查询复用。
type GrpcAuditHandler struct {
	user.UnimplementedAuditServiceServer
	db *gorm.DB
}

func NewGrpcAuditHandler(db *gorm.DB) *GrpcAuditHandler {
	return &GrpcAuditHandler{db: db}
}

func (h *GrpcAuditHandler) RecordLog(ctx context.Context, req *user.RecordLogRequest) (*user.RecordLogResponse, error) {
	log := &model.OperationLog{
		OperatorID:  uint(req.OperatorId),
		Operator:    req.Operator,
		Action:      req.Action,
		TargetType:  req.TargetType,
		TargetID:    uint(req.TargetId),
		TargetTitle: req.TargetTitle,
		Detail:      req.Detail,
		IP:          req.Ip,
	}
	if err := audit.Record(ctx, h.db, log); err != nil {
		return &user.RecordLogResponse{
			Code:    1,
			Message: err.Error(),
		}, nil
	}
	return &user.RecordLogResponse{
		Code:    0,
		Message: "success",
		Id:      uint32(log.ID),
	}, nil
}

func (h *GrpcAuditHandler) ListLogs(ctx context.Context, req *user.ListLogsRequest) (*user.ListLogsResponse, error) {
	logs, total, err := audit.List(ctx, h.db, int(req.Page), int(req.PageSize), req.Action, req.TargetType, uint(req.OperatorId))
	if err != nil {
		return &user.ListLogsResponse{
			Code:    1,
			Message: err.Error(),
		}, nil
	}
	out := make([]*user.OperationLog, 0, len(logs))
	for _, l := range logs {
		out = append(out, &user.OperationLog{
			Id:          uint32(l.ID),
			OperatorId:  uint32(l.OperatorID),
			Operator:    l.Operator,
			Action:      l.Action,
			TargetType:  l.TargetType,
			TargetId:    uint32(l.TargetID),
			TargetTitle: l.TargetTitle,
			Detail:      l.Detail,
			Ip:          l.IP,
			CreatedAt:   l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &user.ListLogsResponse{
		Code:    0,
		Message: "success",
		Logs:    out,
		Total:   uint32(total),
	}, nil
}
