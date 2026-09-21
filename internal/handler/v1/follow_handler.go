package v1

import (
	"context"

	user "github.com/mysunshines/blog-user/proto/pb/v1"
)

// Follow 关注：参数校验在服务层完成，handler 负责透传并在成功后回填双方计数。
func (h *GrpcUserHandler) Follow(ctx context.Context, req *user.FollowRequest) (*user.FollowResponse, error) {
	err := h.Svc.Follow(ctx, uint(req.FollowerId), uint(req.FollowingId))
	if err != nil {
		return &user.FollowResponse{
			Code:    uint32(user.UserErrorCode_USER_UPDATE_FAILED),
			Message: err.Error(),
		}, nil
	}
	_, followerFollowing, _ := h.Svc.GetFollowStats(ctx, uint(req.FollowerId))
	followedFollowers, _, _ := h.Svc.GetFollowStats(ctx, uint(req.FollowingId))
	return &user.FollowResponse{
		Code:           uint32(user.UserErrorCode_USER_SUCCESS),
		Message:        "success",
		FollowerCount:  uint32(followerFollowing), // 关注者当前关注数
		FollowingCount: uint32(followedFollowers), // 被关注者当前粉丝数
	}, nil
}

// Unfollow 取关
func (h *GrpcUserHandler) Unfollow(ctx context.Context, req *user.UnfollowRequest) (*user.UnfollowResponse, error) {
	err := h.Svc.Unfollow(ctx, uint(req.FollowerId), uint(req.FollowingId))
	if err != nil {
		return &user.UnfollowResponse{
			Code:    uint32(user.UserErrorCode_USER_UPDATE_FAILED),
			Message: err.Error(),
		}, nil
	}
	_, followerFollowing, _ := h.Svc.GetFollowStats(ctx, uint(req.FollowerId))
	followedFollowers, _, _ := h.Svc.GetFollowStats(ctx, uint(req.FollowingId))
	return &user.UnfollowResponse{
		Code:           uint32(user.UserErrorCode_USER_SUCCESS),
		Message:        "success",
		FollowerCount:  uint32(followerFollowing),
		FollowingCount: uint32(followedFollowers),
	}, nil
}

// GetFollowStats 粉丝数 / 关注数（公开只读）
func (h *GrpcUserHandler) GetFollowStats(ctx context.Context, req *user.GetFollowStatsRequest) (*user.GetFollowStatsResponse, error) {
	followers, following, err := h.Svc.GetFollowStats(ctx, uint(req.UserId))
	if err != nil {
		return &user.GetFollowStatsResponse{
			Code:    uint32(user.UserErrorCode_USER_NOT_FOUND),
			Message: err.Error(),
		}, nil
	}
	return &user.GetFollowStatsResponse{
		Code:           uint32(user.UserErrorCode_USER_SUCCESS),
		Message:        "success",
		FollowerCount:  uint32(followers),
		FollowingCount: uint32(following),
	}, nil
}

// GetFollowStatus 当前 follower_id 是否关注 following_id（需登录，follower_id 来自前端登录态）
func (h *GrpcUserHandler) GetFollowStatus(ctx context.Context, req *user.GetFollowStatusRequest) (*user.GetFollowStatusResponse, error) {
	ok, err := h.Svc.GetFollowStatus(ctx, uint(req.FollowerId), uint(req.FollowingId))
	if err != nil {
		return &user.GetFollowStatusResponse{
			Code:    uint32(user.UserErrorCode_USER_NOT_FOUND),
			Message: err.Error(),
		}, nil
	}
	return &user.GetFollowStatusResponse{
		Code:        uint32(user.UserErrorCode_USER_SUCCESS),
		Message:     "success",
		IsFollowing: ok,
	}, nil
}
