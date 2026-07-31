package v1

import (
	"context"

	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/service"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/constants"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/sony/gobreaker"
)

// GrpcUserHandler gRPC 用户处理器
type GrpcUserHandler struct {
	user.UnimplementedUserServiceServer
	Svc service.UserService
	Cb  *gobreaker.CircuitBreaker
}

func (h *GrpcUserHandler) Register(ctx context.Context, req *user.RegisterRequest) (*user.RegisterResponse, error) {
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
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
		User:    ConvertToProtoUser(result.User),
		Token:   result.Token,
	}, nil
}

func (h *GrpcUserHandler) Login(ctx context.Context, req *user.LoginRequest) (*user.LoginResponse, error) {
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
		Code:    uint32(user.UserErrorCode_USER_SUCCESS),
		Message: "success",
		User:    ConvertToProtoUser(result.User),
		Token:   result.Token,
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
			Code:    constants.ErrCodeInternal,
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
			Code:    constants.ErrCodeInternal,
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
			Code:    constants.ErrCodeInternal,
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
			Code:    constants.ErrCodeInternal,
			Message: err.Error(),
		}, nil
	}

	return &user.IsBlacklistResponse{
		Code:        0,
		Message:     "success",
		InBlacklist: inBlacklist,
	}, nil
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
