package repository

import (
	"context"

	"github.com/mysunshines/blog-user/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FollowRepository 关注关系仓储接口
type FollowRepository interface {
	// Follow 关注（幂等：已关注则忽略）。follower_id 关注 following_id。
	Follow(ctx context.Context, followerID, followingID uint) error
	// Unfollow 取关（幂等：未关注则忽略）。
	Unfollow(ctx context.Context, followerID, followingID uint) error
	// IsFollowing 是否已关注。
	IsFollowing(ctx context.Context, followerID, followingID uint) (bool, error)
	// CountFollowers 粉丝数（following_id = userID 的记录数，即被人关注）。
	CountFollowers(ctx context.Context, userID uint) (int64, error)
	// CountFollowing 关注数（follower_id = userID 的记录数，即关注别人）。
	CountFollowing(ctx context.Context, userID uint) (int64, error)
}

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Follow(ctx context.Context, followerID, followingID uint) error {
	// 唯一索引 (follower_id, following_id) + OnConflict DoNothing：并发/重复关注幂等安全。
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.UserFollow{FollowerID: followerID, FollowingID: followingID}).Error
}

func (r *followRepository) Unfollow(ctx context.Context, followerID, followingID uint) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&model.UserFollow{}).Error
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID, followingID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

func (r *followRepository) CountFollowers(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Where("following_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *followRepository) CountFollowing(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserFollow{}).
		Where("follower_id = ?", userID).
		Count(&count).Error
	return count, err
}
