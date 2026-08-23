package v1

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mysunshines/blog-user/internal/audit"
	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/service"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/captcha"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/sony/gobreaker"
	"gorm.io/gorm"
)

// GrpcUserHandler gRPC 用户处理器，承载 UserService（含操作审计 RecordLog/ListOperationLogs）。
type GrpcUserHandler struct {
	user.UnimplementedUserServiceServer
	Svc service.UserService
	Cb  *gobreaker.CircuitBreaker
	DB  *gorm.DB
}

func (h *GrpcUserHandler) Register(ctx context.Context, req *user.RegisterRequest) (*user.RegisterResponse, error) {
	// 图形验证码校验（防机器人批量注册）：校验后即作废，避免重放。
	if ok, err := captcha.Verify(ctx, req.CaptchaId, req.CaptchaCode); err != nil || !ok {
		return &user.RegisterResponse{
			Code:    uint32(user.UserErrorCode_USER_REGISTER_FAILED),
			Message: "验证码错误或已失效",
		}, nil
	}

	result, err := h.Svc.Register(ctx, &model.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		return &user.RegisterResponse{
			Code:    uint32(user.UserErrorCode_USER_REGISTER_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &user.RegisterResponse{
		Code:      uint32(user.UserErrorCode_USER_SUCCESS),
		Message:   "success",
		User:      ConvertToProtoUser(result.User),
		Token:     result.Token,
		CsrfToken: result.CSRFToken,
	}, nil
}

func (h *GrpcUserHandler) Login(ctx context.Context, req *user.LoginRequest) (*user.LoginResponse, error) {
	// 图形验证码校验（防撞库/暴力破解机器人）。
	if ok, err := captcha.Verify(ctx, req.CaptchaId, req.CaptchaCode); err != nil || !ok {
		return &user.LoginResponse{
			Code:    uint32(user.UserErrorCode_USER_LOGIN_FAILED),
			Message: "验证码错误或已失效",
		}, nil
	}

	result, err := h.Svc.Login(ctx, &model.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return &user.LoginResponse{
			Code:    uint32(user.UserErrorCode_USER_LOGIN_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &user.LoginResponse{
		Code:      uint32(user.UserErrorCode_USER_SUCCESS),
		Message:   "success",
		User:      ConvertToProtoUser(result.User),
		Token:     result.Token,
		CsrfToken: result.CSRFToken,
	}, nil
}

// GenerateCaptcha 生成图形验证码，供前端注册/登录页展示。
func (h *GrpcUserHandler) GenerateCaptcha(ctx context.Context, req *user.GenerateCaptchaRequest) (*user.GenerateCaptchaResponse, error) {
	id, img, err := captcha.Generate(ctx)
	if err != nil {
		return &user.GenerateCaptchaResponse{
			Code:    uint32(user.UserErrorCode_USER_REGISTER_FAILED),
			Message: "验证码生成失败: " + err.Error(),
		}, nil
	}
	return &user.GenerateCaptchaResponse{
		Code:        uint32(user.UserErrorCode_USER_SUCCESS),
		Message:     "success",
		CaptchaId:   id,
		ImageBase64: img,
	}, nil
}

func (h *GrpcUserHandler) Logout(ctx context.Context, req *user.LogoutRequest) (*user.LogoutResponse, error) {
	err := h.Svc.Logout(ctx, &model.LogoutRequest{
		UserID: uint(req.UserId),
		Token:  req.Token,
	})
	if err != nil {
		return &user.LogoutResponse{
			Code:    uint32(user.UserErrorCode_USER_UPDATE_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &user.LogoutResponse{
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
	}, nil
}

func (h *GrpcUserHandler) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	result, err := h.Svc.GetUser(ctx, uint(req.UserId), req.Username)
	if err != nil {
		return &user.GetUserResponse{
			Code:    uint32(user.UserErrorCode_USER_NOT_FOUND),
			Message: err.Error(),
		}, nil
	}

	return &user.GetUserResponse{
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
		User:    ConvertToProtoUser(result),
	}, nil
}

func (h *GrpcUserHandler) ValidateToken(ctx context.Context, req *user.ValidateTokenRequest) (*user.ValidateTokenResponse, error) {
	result, err := h.Svc.ValidateToken(ctx, req.Token)
	if err != nil {
		return &user.ValidateTokenResponse{
			Code:    uint32(user.UserErrorCode_USER_TOKEN_INVALID),
			Message: err.Error(),
			Valid:   false,
		}, nil
	}

	return &user.ValidateTokenResponse{
		Code:     uint32(user.UserErrorCode_USER_SUCCESS),
		Message:  "success",
		UserId:   uint32(result.UserID),
		Username: result.Username,
		Valid:    result.Valid,
	}, nil
}

func (h *GrpcUserHandler) UpdateUser(ctx context.Context, req *user.UpdateUserRequest) (*user.UpdateUserResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.Svc.UpdateUser(ctx, &model.UpdateUserRequest{
		UserID:   uid,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Bio:      req.Bio,
	})
	if err != nil {
		return &user.UpdateUserResponse{
			Code:    uint32(user.UserErrorCode_USER_UPDATE_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &user.UpdateUserResponse{
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
		User:    ConvertToProtoUser(result),
	}, nil
}

func (h *GrpcUserHandler) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*user.DeleteUserResponse, error) {
	if _, err := commonmiddleware.RequireGRPCAuth(ctx); err != nil {
		return nil, err
	}
	err := h.Svc.DeleteUser(ctx, uint(req.UserId))
	if err != nil {
		return &user.DeleteUserResponse{
			Code:    uint32(user.UserErrorCode_USER_DELETE_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &user.DeleteUserResponse{
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
	}, nil
}

func (h *GrpcUserHandler) GetUsers(ctx context.Context, req *user.GetUsersRequest) (*user.GetUsersResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}

	users, total, err := h.Svc.GetUsers(ctx, &model.GetUsersRequest{
		Page:     page,
		PageSize: pageSize,
		Role:     uint8(req.Role),
	})
	if err != nil {
		return &user.GetUsersResponse{
			Code:    uint32(user.UserErrorCode_USER_INTERNAL_ERROR),
			Message: err.Error(),
		}, nil
	}

	protoUsers := make([]*user.User, len(users))
	for i, u := range users {
		protoUsers[i] = ConvertToProtoUser(u)
	}

	return &user.GetUsersResponse{
		Code:    0,
		Message: "success",
		Users:   protoUsers,
		Total:   uint32(total),
	}, nil
}

func (h *GrpcUserHandler) ChangePassword(ctx context.Context, req *user.ChangePasswordRequest) (*user.ChangePasswordResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	err = h.Svc.ChangePassword(ctx, &model.ChangePasswordRequest{
		UserID:      uid,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return &user.ChangePasswordResponse{
			Code:    uint32(user.UserErrorCode_USER_PASSWORD_INCORRECT),
			Message: err.Error(),
		}, nil
	}

	return &user.ChangePasswordResponse{
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
	}, nil
}

func (h *GrpcUserHandler) AddToBlacklist(ctx context.Context, req *user.BlacklistRequest) (*user.BlacklistResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	err = h.Svc.AddToBlacklist(ctx, &model.BlacklistRequest{
		UserID:       uid,
		TargetUserID: uint(req.TargetUserId),
		Reason:       req.Reason,
	})
	if err != nil {
		return &user.BlacklistResponse{
			Code:    uint32(user.UserErrorCode_USER_INTERNAL_ERROR),
			Message: err.Error(),
		}, nil
	}

	return &user.BlacklistResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (h *GrpcUserHandler) RemoveFromBlacklist(ctx context.Context, req *user.BlacklistRequest) (*user.BlacklistResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	err = h.Svc.RemoveFromBlacklist(ctx, &model.BlacklistRequest{
		UserID:       uid,
		TargetUserID: uint(req.TargetUserId),
		Reason:       req.Reason,
	})
	if err != nil {
		return &user.BlacklistResponse{
			Code:    uint32(user.UserErrorCode_USER_INTERNAL_ERROR),
			Message: err.Error(),
		}, nil
	}

	return &user.BlacklistResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (h *GrpcUserHandler) IsInBlacklist(ctx context.Context, req *user.IsBlacklistRequest) (*user.IsBlacklistResponse, error) {
	inBlacklist, err := h.Svc.IsInBlacklist(ctx, uint(req.UserId), uint(req.TargetUserId))
	if err != nil {
		return &user.IsBlacklistResponse{
			Code:    uint32(user.UserErrorCode_USER_INTERNAL_ERROR),
			Message: err.Error(),
		}, nil
	}

	return &user.IsBlacklistResponse{
		Code:        0,
		Message:     "success",
		InBlacklist: inBlacklist,
	}, nil
}

// requireGRPCAdmin 校验调用方已登录且具备管理员角色。
// 鉴权失败直接返回 gRPC 标准错误，由 Gateway 转换为 4xx。
func requireGRPCAdmin(ctx context.Context) error {
	if _, err := commonmiddleware.RequireGRPCAuth(ctx); err != nil {
		return err
	}
	raw, ok := commonmiddleware.GetGRPCRole(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "未认证")
	}
	var role uint8
	switch v := raw.(type) {
	case float64:
		role = uint8(v)
	case uint8:
		role = v
	case int:
		role = uint8(v)
	case int64:
		role = uint8(v)
	default:
		return status.Error(codes.PermissionDenied, "无效的角色信息")
	}
	if role != model.UserRoleAdmin {
		return status.Error(codes.PermissionDenied, "需要管理员权限")
	}
	return nil
}

// ConvertToProtoUser 将 model.UserResponse 转换为 proto.User
func ConvertToProtoUser(u *model.UserResponse) *user.User {
	if u == nil {
		return nil
	}
	return &user.User{
		Id:        uint32(u.ID),
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		Role:      uint32(u.Role),
		Status:    uint32(u.Status),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// RecordLog 实现 user.v1.UserService，作为各业务服务上报操作日志的汇聚点。
// article-service / comment-service 等通过 gRPC 调用此接口写入 operation_logs。
func (h *GrpcUserHandler) RecordLog(ctx context.Context, req *user.RecordLogRequest) (*user.RecordLogResponse, error) {
	log := &model.OperationLog{
		OperatorID:  uint(req.OperatorId),
		Operator:    req.Operator,
		Action:      audit.ActionToShort(req.Action),
		TargetType:  req.TargetType,
		TargetID:    uint(req.TargetId),
		TargetTitle: req.TargetTitle,
		Detail:      req.Detail,
		IP:          req.Ip,
	}
	if err := audit.Record(ctx, h.DB, log); err != nil {
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

// ListOperationLogs 实现 user.v1.UserService，供管理端查询操作日志（仅管理员）。
func (h *GrpcUserHandler) ListOperationLogs(ctx context.Context, req *user.ListOperationLogsRequest) (*user.ListOperationLogsResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	logs, total, err := audit.List(ctx, h.DB, int(req.Page), int(req.PageSize), req.Action, req.TargetType, uint(req.OperatorId))
	if err != nil {
		return &user.ListOperationLogsResponse{
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
	return &user.ListOperationLogsResponse{
		Code:    0,
		Message: "success",
		Logs:    out,
		Total:   uint32(total),
	}, nil
}

// AdminGetUsers 管理端用户列表（仅管理员）。
func (h *GrpcUserHandler) AdminGetUsers(ctx context.Context, req *user.AdminGetUsersRequest) (*user.AdminGetUsersResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	usersReq := &model.GetUsersRequest{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		Role:     uint8(req.Role),
	}
	if req.Status != 0 {
		statusVal := uint8(req.Status)
		usersReq.Status = &statusVal
	}
	users, total, err := h.Svc.AdminGetUsers(ctx, usersReq)
	if err != nil {
		return &user.AdminGetUsersResponse{
			Code:    1,
			Message: err.Error(),
		}, nil
	}
	protoUsers := make([]*user.User, len(users))
	for i, u := range users {
		protoUsers[i] = ConvertToProtoUser(u)
	}
	return &user.AdminGetUsersResponse{
		Code:    0,
		Message: "success",
		Users:   protoUsers,
		Total:   uint32(total),
	}, nil
}

// AdminUpdateUser 管理端更新用户（角色/状态/昵称，仅管理员）。
func (h *GrpcUserHandler) AdminUpdateUser(ctx context.Context, req *user.AdminUpdateUserRequest) (*user.AdminUpdateUserResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	operatorID, _ := commonmiddleware.GetGRPCUserID(ctx)
	targetID := uint(req.Id)

	updateReq := &model.AdminUpdateUserRequest{
		UserID:   targetID,
		Nickname: req.Nickname,
	}
	if req.Role != 0 {
		role := uint8(req.Role)
		updateReq.Role = &role
	}
	if req.Status != 0 {
		statusVal := uint8(req.Status)
		updateReq.Status = &statusVal
	}

	updated, err := h.Svc.AdminUpdateUser(ctx, updateReq)
	if err != nil {
		return &user.AdminUpdateUserResponse{
			Code:    1,
			Message: err.Error(),
		}, nil
	}

	// 审计：记录管理端对用户资料的改动（失败仅告警，不影响主流程）
	detail := "nickname=" + req.Nickname
	if req.Role != 0 {
		detail += ";role=" + string(rune('0'+req.Role))
	}
	if req.Status != 0 {
		detail += ";status=" + string(rune('0'+req.Status))
	}
	if err := audit.Record(ctx, h.DB, &model.OperationLog{
		OperatorID: operatorID,
		Action:     "update_user",
		TargetType: "user",
		TargetID:   targetID,
		Detail:     detail,
	}); err != nil {
		log.Printf("[audit] user admin record failed: %v", err)
	}

	return &user.AdminUpdateUserResponse{
		Code:    0,
		Message: "success",
		User:    ConvertToProtoUser(updated),
	}, nil
}

// AdminDeleteUser 管理端删除用户（仅管理员）。
func (h *GrpcUserHandler) AdminDeleteUser(ctx context.Context, req *user.AdminDeleteUserRequest) (*user.DeleteUserResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	operatorID, _ := commonmiddleware.GetGRPCUserID(ctx)
	targetID := uint(req.Id)

	if err := h.Svc.AdminDeleteUser(ctx, targetID); err != nil {
		return &user.DeleteUserResponse{
			Code:    1,
			Message: err.Error(),
		}, nil
	}

	// 审计：记录管理端删除操作（失败仅告警）
	if err := audit.Record(ctx, h.DB, &model.OperationLog{
		OperatorID: operatorID,
		Action:     "delete_user",
		TargetType: "user",
		TargetID:   targetID,
		Detail:     "admin delete",
	}); err != nil {
		log.Printf("[audit] user admin record failed: %v", err)
	}

	return &user.DeleteUserResponse{
		Code:    0,
		Message: "success",
	}, nil
}
