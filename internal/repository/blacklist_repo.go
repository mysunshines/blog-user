package repository

import (
	"context"
	"fmt"

	"github.com/mysunshines/blog-user/internal/errors"
	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/gocommon/cache"

	"gorm.io/gorm"
)

type BlacklistRepository interface {
	AddToBlacklist(ctx context.Context, userID, blockedUserID uint, reason string) error
	RemoveFromBlacklist(ctx context.Context, userID, blockedUserID uint) error
	IsInBlacklist(ctx context.Context, userID, blockedUserID uint) (bool, error)
	GetUserBlacklist(ctx context.Context, userID uint) ([]*model.UserBlacklist, error)
}

type blacklistRepository struct {
	db *gorm.DB
}

func NewBlacklistRepository(db *gorm.DB) BlacklistRepository {
	return &blacklistRepository{
		db: db,
	}
}

func (r *blacklistRepository) AddToBlacklist(ctx context.Context, userID, blockedUserID uint, reason string) error {
	blacklist := &model.UserBlacklist{
		UserID:        userID,
		BlockedUserID: blockedUserID,
		Reason:        reason,
	}

	err := r.db.WithContext(ctx).Create(blacklist).Error
	if err != nil {
		return errors.Internal("blacklist failed", err)
	}

	// 更新缓存
	key := getBlacklistKey(userID)
	cache.SAdd(ctx, key, blockedUserID)

	return nil
}

func (r *blacklistRepository) RemoveFromBlacklist(ctx context.Context, userID, blockedUserID uint) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Delete(&model.UserBlacklist{})

	if result.Error != nil {
		return errors.Internal("blacklist failed", result.Error)
	}

	// 更新缓存
	key := getBlacklistKey(userID)
	cache.SRem(ctx, key, blockedUserID)

	return nil
}

func (r *blacklistRepository) IsInBlacklist(ctx context.Context, userID, blockedUserID uint) (bool, error) {
	// 先检查缓存
	key := getBlacklistKey(userID)
	exists, err := cache.SIsMember(ctx, key, blockedUserID)
	if err == nil && exists {
		return true, nil
	}

	// 缓存未命中，查询数据库
	var count int64
	result := r.db.WithContext(ctx).Model(&model.UserBlacklist{}).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Count(&count)

	if result.Error != nil {
		return false, errors.Internal("blacklist failed", result.Error)
	}

	// 更新缓存
	if count > 0 {
		cache.SAdd(ctx, key, blockedUserID)
	}

	return count > 0, nil
}

func (r *blacklistRepository) GetUserBlacklist(ctx context.Context, userID uint) ([]*model.UserBlacklist, error) {
	var blacklists []*model.UserBlacklist
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&blacklists)

	if result.Error != nil {
		return nil, errors.Internal("blacklist failed", result.Error)
	}

	return blacklists, nil
}

func getBlacklistKey(userID uint) string {
	return fmt.Sprintf("blacklist:%d", userID)
}
