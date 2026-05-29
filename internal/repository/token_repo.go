package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/mysunshines/blog-user/pkg/errors"
	"github.com/mysunshines/gocommon/cache"

	"gorm.io/gorm"
)

type TokenRepository interface {
	Create(ctx context.Context, token *TokenRecord) error
	GetByToken(ctx context.Context, token string) (*TokenRecord, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uint) error
	IsInBlacklist(ctx context.Context, token string) (bool, error)
	AddToBlacklist(ctx context.Context, token string, expireTime time.Time) error
}

type TokenRecord struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	Token     string    `gorm:"size:512;not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null;index"`
	CreatedAt time.Time
}

func (TokenRecord) TableName() string {
	return "user_tokens"
}

type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{
		db: db,
	}
}

func (r *tokenRepository) Create(ctx context.Context, token *TokenRecord) error {
	err := r.db.WithContext(ctx).Create(token).Error
	if err != nil {
		return errors.Internal("创建token记录失败", err)
	}
	return nil
}

func (r *tokenRepository) GetByToken(ctx context.Context, token string) (*TokenRecord, error) {
	var record TokenRecord
	result := r.db.WithContext(ctx).Where("token = ?", token).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal("查询token失败", result.Error)
	}
	return &record, nil
}

func (r *tokenRepository) Delete(ctx context.Context, token string) error {
	result := r.db.WithContext(ctx).Where("token = ?", token).Delete(&TokenRecord{})
	if result.Error != nil {
		return errors.Internal("删除token失败", result.Error)
	}
	return nil
}

func (r *tokenRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&TokenRecord{})
	if result.Error != nil {
		return errors.Internal("删除用户token失败", result.Error)
	}
	return nil
}

func (r *tokenRepository) IsInBlacklist(ctx context.Context, token string) (bool, error) {
	key := getTokenBlacklistKey()
	exists, err := cache.SIsMember(ctx, key, token)
	if err != nil {
		return false, errors.Internal("检查token黑名单失败", err)
	}
	return exists, nil
}

func (r *tokenRepository) AddToBlacklist(ctx context.Context, token string, expireTime time.Time) error {
	key := getTokenBlacklistKey()
	ttl := time.Until(expireTime)
	if ttl > 0 {
		// 使用 Sorted Set 存储，自动过期
		_, err := cache.ZAdd(ctx, key, &cache.Z{
			Score:  float64(expireTime.Unix()),
			Member: token,
		})
		if err != nil {
			return errors.Internal("添加token到黑名单失败", err)
		}
		// 定期清理过期token
		cache.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(time.Now().Unix(), 10))
	}
	return nil
}

func getTokenBlacklistKey() string {
	return "token:blacklist"
}
