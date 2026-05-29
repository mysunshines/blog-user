package service

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/mysunshines/blog-user/internal/config"
	"github.com/mysunshines/blog-user/internal/model"
	"github.com/mysunshines/blog-user/internal/repository"
	apperrors "github.com/mysunshines/blog-user/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mysunshines/gocommon/log"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务接口
type UserService interface {
	// 认证相关
	Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error)
	Logout(ctx context.Context, req *model.LogoutRequest) error
	ValidateToken(ctx context.Context, token string) (*ValidateTokenResult, error)

	// 用户操作
	GetUser(ctx context.Context, userID uint, username string) (*model.UserResponse, error)
	UpdateUser(ctx context.Context, req *model.UpdateUserRequest) (*model.UserResponse, error)
	DeleteUser(ctx context.Context, userID uint) error
	GetUsers(ctx context.Context, req *model.GetUsersRequest) ([]*model.UserResponse, int64, error)
	ChangePassword(ctx context.Context, req *model.ChangePasswordRequest) error

	// 黑名单操作
	AddToBlacklist(ctx context.Context, req *model.BlacklistRequest) error
	RemoveFromBlacklist(ctx context.Context, req *model.BlacklistRequest) error
	IsInBlacklist(ctx context.Context, userID, targetUserID uint) (bool, error)
}

// ValidateTokenResult Token验证结果
type ValidateTokenResult struct {
	UserID   uint
	Username string
	Valid    bool
}

// userService 用户服务实现
type userService struct {
	repo repository.UserRepository
	cfg  *config.Config
	bf   *BloomFilter
}

// BloomFilter 布隆过滤器
type BloomFilter struct {
	data map[string]bool
}

// NewBloomFilter 创建布隆过滤器
func NewBloomFilter(size uint) *BloomFilter {
	return &BloomFilter{
		data: make(map[string]bool),
	}
}

// Add 添加元素
func (bf *BloomFilter) Add(key string) {
	bf.data[key] = true
}

// Contains 检查元素是否存在
func (bf *BloomFilter) Contains(key string) bool {
	return bf.data[key]
}

// NewUserService 创建用户服务
func NewUserService(repo repository.UserRepository, cfg *config.Config) UserService {
	return &userService{
		repo: repo,
		cfg:  cfg,
		bf:   NewBloomFilter(100000),
	}
}

// Register 用户注册
func (s *userService) Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error) {
	// 检查用户名和邮箱格式
	if !isValidUsername(req.Username) {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid username format")
	}
	if !isValidEmail(req.Email) {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid email format")
	}
	if !isValidPassword(req.Password) {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Password must contain letters and numbers, 6-32 characters")
	}

	// 检查用户是否已存在
	exists, err := s.repo.ExistsByUsernameOrEmail(ctx, req.Username, req.Email)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrRegisterFailed, "Failed to check user existence")
	}
	if exists {
		return nil, apperrors.New(apperrors.ErrUserAlreadyExists, "User already exists")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrRegisterFailed, "Failed to hash password")
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Avatar:   generateAvatar(req.Username),
		Role:     model.UserRoleNormal,
		Status:   model.UserStatusNormal,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, apperrors.New(apperrors.ErrRegisterFailed, "Failed to create user")
	}

	// 添加到布隆过滤器
	s.bf.Add(req.Username)

	// 生成Token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrRegisterFailed, "Failed to generate token")
	}

	// 保存Token
	if err := s.saveToken(ctx, user.ID, token); err != nil {
		log.Warnf("Failed to save token for user %d: %v", user.ID, err)
	}

	return &model.AuthResponse{
		Token: token,
		User:  user.ToResponsePtr(),
	}, nil
}

// Login 用户登录
func (s *userService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	// 获取用户
	user, err := s.repo.GetByUsernameOrEmail(ctx, req.Username, req.Username)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.New(apperrors.ErrUserAlreadyExists, "Invalid username or password")
		}
		return nil, apperrors.New(apperrors.ErrLoginFailed, "Failed to get user")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, apperrors.New(apperrors.ErrPasswordIncorrect, "Invalid username or password")
	}

	// 检查用户状态
	if user.Status != model.UserStatusNormal {
		return nil, apperrors.New(apperrors.ErrForbidden, "User account is disabled")
	}

	// 生成Token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrLoginFailed, "Failed to generate token")
	}

	// 保存Token
	if err := s.saveToken(ctx, user.ID, token); err != nil {
		log.Warnf("Failed to save token for user %d: %v", user.ID, err)
	}

	return &model.AuthResponse{
		Token: token,
		User:  user.ToResponsePtr(),
	}, nil
}

// Logout 用户登出
func (s *userService) Logout(ctx context.Context, req *model.LogoutRequest) error {
	// 删除Token
	if err := s.repo.DeleteToken(ctx, req.Token); err != nil {
		return apperrors.New(apperrors.ErrInternal, "Failed to logout")
	}
	return nil
}

// ValidateToken 验证Token
func (s *userService) ValidateToken(ctx context.Context, token string) (*ValidateTokenResult, error) {
	// 解析JWT
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, stderrors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil || !parsedToken.Valid {
		return &ValidateTokenResult{Valid: false}, nil
	}

	// 获取Token记录
	tokenRecord, err := s.repo.GetToken(ctx, token)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return &ValidateTokenResult{Valid: false}, nil
		}
		return &ValidateTokenResult{Valid: false}, nil
	}

	// 检查Token是否过期
	if time.Now().After(tokenRecord.ExpiresAt) {
		return &ValidateTokenResult{Valid: false}, nil
	}

	// 获取用户信息
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return &ValidateTokenResult{Valid: false}, nil
	}

	userID := uint(claims["user_id"].(float64))
	username := claims["username"].(string)

	return &ValidateTokenResult{
		UserID:   userID,
		Username: username,
		Valid:    true,
	}, nil
}

// GetUser 获取用户信息
func (s *userService) GetUser(ctx context.Context, userID uint, username string) (*model.UserResponse, error) {
	var user *model.User
	var err error

	if userID > 0 {
		user, err = s.repo.GetByID(ctx, userID)
	} else if username != "" {
		user, err = s.repo.GetByUsername(ctx, username)
	} else {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "user_id or username required")
	}

	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.New(apperrors.ErrUserNotFound, "User not found")
		}
		return nil, apperrors.New(apperrors.ErrInternal, "Failed to get user")
	}

	return user.ToResponsePtr(), nil
}

// UpdateUser 更新用户信息
func (s *userService) UpdateUser(ctx context.Context, req *model.UpdateUserRequest) (*model.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return nil, apperrors.New(apperrors.ErrUserNotFound, "User not found")
		}
		return nil, apperrors.New(apperrors.ErrInternal, "Failed to get user")
	}

	// 更新字段
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, apperrors.New(apperrors.ErrUpdateFailed, "Failed to update user")
	}

	return user.ToResponsePtr(), nil
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(ctx context.Context, userID uint) error {
	// 删除用户的Token
	if err := s.repo.DeleteUserTokens(ctx, userID); err != nil {
		log.Warnf("Failed to delete tokens for user %d: %v", userID, err)
	}

	// 删除用户
	if err := s.repo.Delete(ctx, userID); err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return apperrors.New(apperrors.ErrUserNotFound, "User not found")
		}
		return apperrors.New(apperrors.ErrDeleteFailed, "Failed to delete user")
	}

	return nil
}

// GetUsers 获取用户列表
func (s *userService) GetUsers(ctx context.Context, req *model.GetUsersRequest) ([]*model.UserResponse, int64, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	users, total, err := s.repo.List(ctx, page, pageSize, req.Role)
	if err != nil {
		return nil, 0, apperrors.New(apperrors.ErrInternal, "Failed to list users")
	}

	responses := make([]*model.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponsePtr()
	}

	return responses, total, nil
}

// ChangePassword 修改密码
func (s *userService) ChangePassword(ctx context.Context, req *model.ChangePasswordRequest) error {
	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		if stderrors.Is(err, repository.ErrNotFound) {
			return apperrors.New(apperrors.ErrUserNotFound, "User not found")
		}
		return apperrors.New(apperrors.ErrInternal, "Failed to get user")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return apperrors.New(apperrors.ErrPasswordIncorrect, "Old password incorrect")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "Failed to hash password")
	}

	// 更新密码
	user.Password = string(hashedPassword)
	if err := s.repo.Update(ctx, user); err != nil {
		return apperrors.New(apperrors.ErrUpdateFailed, "Failed to update password")
	}

	return nil
}

// AddToBlacklist 添加到黑名单
func (s *userService) AddToBlacklist(ctx context.Context, req *model.BlacklistRequest) error {
	blacklist := &model.UserBlacklist{
		UserID:        req.UserID,
		BlockedUserID: req.TargetUserID,
		Reason:        req.Reason,
	}

	if err := s.repo.AddToBlacklist(ctx, blacklist); err != nil {
		return apperrors.New(apperrors.ErrInternal, "Failed to add to blacklist")
	}

	return nil
}

// RemoveFromBlacklist 从黑名单移除
func (s *userService) RemoveFromBlacklist(ctx context.Context, req *model.BlacklistRequest) error {
	if err := s.repo.RemoveFromBlacklist(ctx, req.UserID, req.TargetUserID); err != nil {
		return apperrors.New(apperrors.ErrInternal, "Failed to remove from blacklist")
	}
	return nil
}

// IsInBlacklist 检查是否在黑名单中
func (s *userService) IsInBlacklist(ctx context.Context, userID, targetUserID uint) (bool, error) {
	inBlacklist, err := s.repo.IsInBlacklist(ctx, userID, targetUserID)
	if err != nil {
		return false, apperrors.New(apperrors.ErrInternal, "Failed to check blacklist")
	}
	return inBlacklist, nil
}

// generateToken 生成JWT Token
func (s *userService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Duration(s.cfg.JWT.ExpireTime) * time.Second).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// saveToken 保存Token
func (s *userService) saveToken(ctx context.Context, userID uint, token string) error {
	tokenRecord := &model.Token{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(s.cfg.JWT.ExpireTime) * time.Second),
	}
	return s.repo.CreateToken(ctx, tokenRecord)
}

// ============ 辅助函数 ============

// isValidUsername 验证用户名
func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// isValidEmail 验证邮箱
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 128 {
		return false
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // 多个@
			}
			atIndex = i
		}
	}
	return atIndex > 0 && atIndex < len(email)-1
}

// isValidPassword 验证密码
func isValidPassword(password string) bool {
	if len(password) < 6 || len(password) > 32 {
		return false
	}
	hasLetter := false
	hasNumber := false
	for _, c := range password {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasNumber = true
		}
	}
	return hasLetter && hasNumber
}

// generateAvatar 生成默认头像
func generateAvatar(username string) string {
	return "https://api.dicebear.com/7.x/avataaars/svg?seed=" + username
}
