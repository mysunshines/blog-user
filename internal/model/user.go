package model

import (
	"time"

	"github.com/mysunshines/gocommon/constants"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Email     string    `gorm:"uniqueIndex;size:128;not null" json:"email"`
	Password  string    `gorm:"size:128;not null" json:"-"`
	Nickname  string    `gorm:"size:64" json:"nickname"`
	Avatar    string    `gorm:"size:256" json:"avatar"`
	Bio       string    `gorm:"size:512" json:"bio"`
	Role      uint8     `gorm:"default:1" json:"role"`   // 1: 普通用户 2: 管理员
	Status    uint8     `gorm:"default:1" json:"status"` // 1: 正常 0: 禁用
	CreatedAt time.Time  `gorm:"<-:create" json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// Token Token模型
type Token struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Token     string    `gorm:"size:512;not null" json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (Token) TableName() string {
	return "tokens"
}

// UserBlacklist 用户黑名单
type UserBlacklist struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`         // 用户ID
	BlockedUserID uint      `gorm:"index;not null" json:"blocked_user_id"` // 被拉黑的用户ID
	Reason        string    `gorm:"size:256" json:"reason"`                // 拉黑原因
	CreatedAt     time.Time `json:"created_at"`
}

func (UserBlacklist) TableName() string {
	return "user_blacklists"
}

// UserStatus 用户状态常量
const (
	UserStatusNormal   uint8 = 1 // 正常
	UserStatusDisabled uint8 = 0 // 禁用
)

// UserRole 用户角色常量
const (
	UserRoleNormal uint8 = 1 // 普通用户
	UserRoleAdmin  uint8 = 2 // 管理员
)

// ============ 请求结构 ============

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username   string `json:"username" binding:"required,min=2,max=32"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6,max=32"`
	Nickname   string `json:"nickname"`
	VerifyCode string `json:"verify_code"`
}

// SendVerifyCodeRequest 发送验证码请求
type SendVerifyCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Token  string `json:"token" binding:"required"`
}

// GetUserRequest 获取用户请求
type GetUserRequest struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	UserID   uint   `json:"user_id" binding:"required"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
}

// DeleteUserRequest 删除用户请求
type DeleteUserRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// GetUsersRequest 获取用户列表请求
type GetUsersRequest struct {
	Page     int   `form:"page,default=1" binding:"min=1"`
	PageSize int   `form:"page_size,default=20" binding:"min=1,max=100"`
	Role     uint8 `form:"role"`
	Status   *uint8 `form:"status"`
}

// AdminUpdateUserRequest 管理员更新用户请求
// user_id 从 URL 参数绑定，不在 JSON body 中强制要求
type AdminUpdateUserRequest struct {
	UserID   uint   `json:"user_id"`
	Nickname string `json:"nickname"`
	Role     *uint8  `json:"role"`
	Status   *uint8  `json:"status"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	UserID      uint   `json:"user_id" binding:"required"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// BlacklistRequest 黑名单请求
type BlacklistRequest struct {
	UserID       uint   `json:"user_id" binding:"required"`
	TargetUserID uint   `json:"target_user_id" binding:"required"`
	Reason       string `json:"reason"`
}

// ValidateTokenRequest 验证Token请求
type ValidateTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// ============ 响应结构 ============

// UserResponse 用户响应
type UserResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Bio       string `json:"bio"`
	Role      uint8  `json:"role"`
	Status    uint8  `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ToResponse 转换为响应格式
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.Format(constants.DateTimeFormat),
		UpdatedAt: u.UpdatedAt.Format(constants.DateTimeFormat),
	}
}

// ToResponsePtr 转换为响应指针
func (u *User) ToResponsePtr() *UserResponse {
	resp := u.ToResponse()
	return &resp
}

// AuthResponse 认证响应（包含Token）
type AuthResponse struct {
	Token     string        `json:"token"`
	CSRFToken string        `json:"csrf_token"`
	User      *UserResponse `json:"user"`
}
