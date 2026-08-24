package repository

import (
	"context"
	"errors"
	"time"

	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/gocommon/pool"

	"gorm.io/gorm"
)

// ErrNotFound 资源不存在错误
var ErrNotFound = errors.New("record not found")

// UserRepository 用户仓储接口
type UserRepository interface {
	// 用户基础操作
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	UpdateFields(ctx context.Context, id uint, cols map[string]interface{}) error
	Delete(ctx context.Context, id uint) error

	// 用户查询
	GetByUsernameOrEmail(ctx context.Context, username, email string) (*model.User, error)
	List(ctx context.Context, page, pageSize int, role uint8, status *uint8) ([]*model.User, int64, error)
	ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
	// ListUsernamesByPage 按 id 游标分页查询用户名（返回下一页游标 id），
	// 供布隆过滤器启动预热，避免大表一次全量加载到内存
	ListUsernamesByPage(ctx context.Context, afterID, limit uint) ([]string, uint, error)

	// Token操作
	CreateToken(ctx context.Context, token *model.Token) error
	GetToken(ctx context.Context, token string) (*model.Token, error)
	DeleteToken(ctx context.Context, token string) error
	DeleteUserTokens(ctx context.Context, userID uint) error

	// 黑名单操作
	AddToBlacklist(ctx context.Context, blacklist *model.UserBlacklist) error
	RemoveFromBlacklist(ctx context.Context, userID, blockedUserID uint) error
	IsInBlacklist(ctx context.Context, userID, blockedUserID uint) (bool, error)
}

// userRepository 用户仓储实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新用户（只更新非零值字段，不更新 created_at）
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Updates(user).Error
}

// UpdateFields 精确更新用户指定列（用 map 可写入零值，如禁用 status=0）
func (r *userRepository) UpdateFields(ctx context.Context, id uint, cols map[string]interface{}) error {
	cols["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(cols).Error
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// GetByUsernameOrEmail 根据用户名或邮箱获取用户
func (r *userRepository) GetByUsernameOrEmail(ctx context.Context, username, email string) (*model.User, error) {
	var user model.User
	query := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", username, email).
		First(&user)
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, query.Error
	}
	return &user, nil
}

// List 获取用户列表
func (r *userRepository) List(ctx context.Context, page, pageSize int, role uint8, status *uint8) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.User{})
	if role > 0 {
		baseQuery = baseQuery.Where("role = ?", role)
	}
	if status != nil {
		baseQuery = baseQuery.Where("status = ?", *status)
	}

	offset := (page - 1) * pageSize

	// 并行：COUNT + SELECT
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).Count(&total).Error
		},
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).
				Offset(offset).Limit(pageSize).
				Find(&users).Error
		},
	)

	if results[0].Err != nil {
		return nil, 0, results[0].Err
	}
	if results[1].Err != nil {
		return nil, 0, results[1].Err
	}

	return users, total, nil
}

// ExistsByUsernameOrEmail 检查用户名或邮箱是否已存在
func (r *userRepository) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error
	return count > 0, err
}

// ListUsernamesByPage 按 id 游标分页查询用户名（GORM 软删除模型自动排除已删除行）。
// 返回本页用户名列表与下一页游标 id（0 表示无更多），供布隆过滤器启动预热分批灌入。
func (r *userRepository) ListUsernamesByPage(ctx context.Context, afterID, limit uint) ([]string, uint, error) {
	var rows []struct {
		ID       uint
		Username string
	}
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id > ?", afterID).Order("id").Limit(int(limit)).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	names := make([]string, 0, len(rows))
	var nextID uint
	for _, row := range rows {
		names = append(names, row.Username)
		nextID = row.ID
	}
	return names, nextID, nil
}

// CreateToken 创建Token
func (r *userRepository) CreateToken(ctx context.Context, token *model.Token) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// GetToken 获取Token
func (r *userRepository) GetToken(ctx context.Context, token string) (*model.Token, error) {
	var tokenModel model.Token
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&tokenModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &tokenModel, nil
}

// DeleteToken 删除Token
func (r *userRepository) DeleteToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Where("token = ?", token).Delete(&model.Token{}).Error
}

// DeleteUserTokens 删除用户的所有Token
func (r *userRepository) DeleteUserTokens(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.Token{}).Error
}

// AddToBlacklist 添加到黑名单
func (r *userRepository) AddToBlacklist(ctx context.Context, blacklist *model.UserBlacklist) error {
	return r.db.WithContext(ctx).Create(blacklist).Error
}

// RemoveFromBlacklist 从黑名单移除
func (r *userRepository) RemoveFromBlacklist(ctx context.Context, userID, blockedUserID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Delete(&model.UserBlacklist{}).Error
}

// IsInBlacklist 检查是否在黑名单中
func (r *userRepository) IsInBlacklist(ctx context.Context, userID, blockedUserID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserBlacklist{}).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Count(&count).Error
	return count > 0, err
}

// CleanExpiredTokens 清理过期的Token
func (r *userRepository) CleanExpiredTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.Token{}).Error
}
