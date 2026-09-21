package service

import (
	"context"
	stderrors "errors"

	"github.com/mysunshines/blog-user/internal/client"
	"github.com/mysunshines/gocommon/log"
)

// Follow 关注：followerID 关注 followingID（不能关注自己，幂等）。成功后增量推送粉丝榜 +1。
func (s *userService) Follow(ctx context.Context, followerID, followingID uint) error {
	if followerID == 0 || followingID == 0 {
		return stderrors.New("参数无效")
	}
	if followerID == followingID {
		return stderrors.New("不能关注自己")
	}
	// 校验被关注用户存在
	if _, err := s.repo.GetByID(ctx, followingID); err != nil {
		return stderrors.New("被关注用户不存在")
	}
	if err := s.followRepo.Follow(ctx, followerID, followingID); err != nil {
		return err
	}
	// 推送粉丝榜（best-effort）：被关注者粉丝数 +1
	if err := client.RecordFansScore(ctx, followingID, 1); err != nil {
		log.Warnf("record fans score failed on follow: %v", err)
	}
	return nil
}

// Unfollow 取关：幂等，未关注直接成功。成功后增量推送粉丝榜 -1。
func (s *userService) Unfollow(ctx context.Context, followerID, followingID uint) error {
	if followerID == 0 || followingID == 0 {
		return stderrors.New("参数无效")
	}
	existed, err := s.followRepo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if !existed {
		return nil
	}
	if err := s.followRepo.Unfollow(ctx, followerID, followingID); err != nil {
		return err
	}
	// 推送粉丝榜（best-effort）：被关注者粉丝数 -1
	if err := client.RecordFansScore(ctx, followingID, -1); err != nil {
		log.Warnf("record fans score failed on unfollow: %v", err)
	}
	return nil
}

// GetFollowStats 返回某用户的粉丝数（被人关注）与关注数（关注别人）。
func (s *userService) GetFollowStats(ctx context.Context, userID uint) (int64, int64, error) {
	followers, err := s.followRepo.CountFollowers(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	following, err := s.followRepo.CountFollowing(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	return followers, following, nil
}

// GetFollowStatus 当前 followerID 是否关注 followingID。
func (s *userService) GetFollowStatus(ctx context.Context, followerID, followingID uint) (bool, error) {
	return s.followRepo.IsFollowing(ctx, followerID, followingID)
}
